package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/tidwall/gjson"
)

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

	// if no candidates in latest release then error out
	// TODO! CHECK THE OTHER RELEASES!!!!
	if gjson.Get(releaseJson, "0.assets.#").Int() == 0 {
		ansiError("Requested package has no releases available")
		return 1
	}

	// get and filter candidates (release files)
	fmt.Printf("Asking GitHub for files on release %s...", gjson.Get(releaseJson, "0.tag_name").String())
	candidates, err := githubGetCandidates(releaseJson)
	if err != nil {
		lnAnsiError("Couldn't get GitHub release files:", err.Error())
		return 1
	}
	fmt.Println(doneMsg)

	if err := filterCandidates(candidates); err != nil {
		ansiError(err.Error())
		return 1
	}

	if len(candidates) != 1 {
		ansiError("Too many candidates (TODO: let user choose)")
		return 1
	}

	// generate tmp dir
	b64, err := randomBase64(16)
	if err != nil {
		ansiError("Couldn't generate tmp id:", err.Error())
		return 1
	}

	tempDir := "/tmp/yadeb-" + b64

	if err := os.Mkdir(tempDir, 0600); err != nil {
		ansiError("Couldn't create tmp folder:", err.Error())
		return 1
	}

	// bad but it's fine (for now). downlad the remaining candidate
	for _, v := range candidates {
		if err := downloadFile(v, fmt.Sprintf("%s/%s", tempDir, v[strings.LastIndex(v, "/")+1:])); err != nil {
			ansiError("Couldn't download selected candidate", err.Error())
			return 1
		}

		break
	}

	return 0
}

// uses github api to get repo's releases
func githubGetReleases(user, repo string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=50", user, repo), nil)
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
func githubGetCandidates(json string) (map[string]string, error) {
	assetCount := gjson.Get(json, "0.assets.#").Int()
	candidates := make(map[string]string)

	for i := range assetCount {
		assetPath := fmt.Sprintf("0.assets.%d", i)

		candidates[gjson.Get(json, assetPath+".name").String()] = gjson.Get(json, assetPath+".browser_download_url").String()
	}

	return candidates, nil
}
