package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
)

// github-specific install code
func githubCmdInstall(u *url.URL) int {
	// get user and repo
	pathParts := strings.Split(u.Path, "/")
	if len(pathParts) < 3 {
		ansiError("Invalid GitHub repo link (not enough path parts)")
		return 2
	}

	user := pathParts[1]
	repo := pathParts[2]

	if user == "" || repo == "" {
		ansiError("Invalid GitHub repo link (empty user or repo)")
		return 2
	}

	// get releases
	fmt.Printf("Asking GitHub for releases on \"%s/%s\"...", user, repo)
	releaseJson, err := githubGetReleases(user, repo)
	if err != nil {
		lnAnsiError("Couldn't get GitHub releases:", err.Error())
		return 1
	}
	fmt.Println(doneMsg)

	if gjson.Get(releaseJson, "#").Int() == 0 {
		ansiError("Requested package has no releases available")
		return 1
	}

	var (
		candidates map[string]string
		tag        string
		validTag   bool
	)

	// go through releases
	for i := range gjson.Get(releaseJson, "#").Int() {
		// get tag
		tag = gjson.Get(releaseJson, fmt.Sprintf("%d.tag_name", i)).String()

		// check if any assets are available
		if gjson.Get(releaseJson, fmt.Sprintf("%d.assets.#", i)).Int() == 0 {
			fmt.Printf("Skipping release %s: \033[91mno assets available\033[0m\n", tag)
			continue
		}

		// get and filter candidates (release files)
		candidates = githubGetCandidates(releaseJson, i)

		if err := filterCandidates(candidates); err != nil {
			fmt.Printf("Skipping release %s: \033[91m%s\033[0m\n", tag, err.Error())
			continue
		}

		if len(candidates) != 1 {
			fmt.Printf("Skipping release %s: \033[91mtoo many candidates (TODO: let user choose)\033[0m\n", tag)
			continue
		}

		validTag = true
		break
	}

	if !validTag {
		ansiError("No valid release found")
		return 1
	}

	// generate tmp dir
	b64, err := randomBase64(16)
	if err != nil {
		ansiError("Couldn't generate tmp id:", err.Error())
		return 1
	}

	tempDir := "/tmp/yadeb-" + b64

	if err := os.Mkdir(tempDir, 0755); err != nil {
		ansiError("Couldn't create tmp folder:", err.Error())
		return 1
	}

	if err := aptChown(tempDir); err != nil {
		ansiError("Couldn't chown downloaded deb to _apt:", err.Error())
		return 1
	}

	// bad but it's fine (for now). downlad the remaining candidate
	for _, v := range candidates {
		path := fmt.Sprintf("%s/%s", tempDir, filepath.Base(v))

		if err := downloadFile(v, path); err != nil {
			ansiError("Couldn't download selected candidate:", err.Error())
			cleanupDir(tempDir)
			return 1
		}

		fmt.Printf("Marking \"%s/%s\" as installed...", user, repo)
		if err := markAsInstalled(path, u.String(), tag); err != nil {
			lnAnsiError(fmt.Sprintf("Couldn't mark %s/%s as installed:", user, repo), err.Error())
			cleanupDir(tempDir)
			return 1
		}
		fmt.Println(doneMsg)

		fmt.Print("Starting APT...\n\n")
		if err := runApt("install", path); err != nil {
			ansiError("Couldn't run APT:", err.Error())
		}
	}

	return cleanupDir(tempDir)
}

// uses github api to get repo's releases
func githubGetReleases(user, repo string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d", user, repo, releaseDepth), nil)
	if err != nil {
		return "", err
	}

	// set headers
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// use githubGetReleases to get the json
func githubGetCandidates(json string, release int64) map[string]string {
	assetCount := gjson.Get(json, fmt.Sprintf("%d.assets.#", release)).Int()
	candidates := make(map[string]string)

	for i := range assetCount {
		assetPath := fmt.Sprintf("%d.assets.%d", release, i)
		candidates[gjson.Get(json, assetPath+".name").String()] = gjson.Get(json, assetPath+".browser_download_url").String()
	}

	return candidates
}
