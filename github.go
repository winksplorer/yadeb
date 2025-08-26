package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/tidwall/gjson"
	"gopkg.in/ini.v1"
)

// github-specific candidate collection
func githubGetCandidates(u *url.URL, tagFlag string, cfg *ini.File) (*map[string]string, string, string, error) {
	// get user and repo
	pathParts := strings.Split(u.Path, "/")
	if len(pathParts) < 3 {
		return nil, "", "", fmt.Errorf("invalid github repo link (not enough path parts)")
	}

	user := pathParts[1]
	repo := pathParts[2]

	if user == "" || repo == "" {
		return nil, "", "", fmt.Errorf("invalid github repo link (empty username or repo name)")
	}

	pkgName := fmt.Sprintf("%s/%s", user, repo)

	// get releases
	fmt.Printf("Asking GitHub for releases on %s...", pkgName)
	releaseJson, err := githubGetReleases(pkgName, cfg.Section("yadeb").Key("ReleaseDepth").MustInt(50))
	if err != nil {
		fmt.Println() // Yes, this is bad. Yes, you will see this alot.
		return nil, "", "", fmt.Errorf("couldn't get github releases: %s", err)
	}
	fmt.Println(doneMsg)

	if gjson.Get(releaseJson, "#").Int() == 0 {
		return nil, "", "", fmt.Errorf("requested package has no releases available")
	}

	var (
		candidates map[string]string
		tag        string
		validTag   bool
	)

	// go through releases
	if tagFlag == "latest" {
		for i := range gjson.Get(releaseJson, "#").Int() {
			// get tag
			tag = gjson.Get(releaseJson, fmt.Sprintf("%d.tag_name", i)).String()

			if !cfg.Section("yadeb").Key("AllowPrerelease").MustBool(false) && gjson.Get(releaseJson, fmt.Sprintf("%d.prerelease", i)).Bool() {
				fmt.Printf("Skipping release %s: \033[91mrelease is a prerelease, which is disallowed\033[0m\n", tag)
				continue
			}

			if err := githubFormatCandidates(&candidates, releaseJson, i); err != nil {
				fmt.Printf("Skipping release %s: \033[91m%s\033[0m\n", tag, err.Error())
				continue
			}

			validTag = true
			break
		}

		if !validTag {
			return nil, "", "", fmt.Errorf("no valid release found")
		}
	} else {
		foundTag, index := githubTagSearch(releaseJson, tagFlag)
		if !foundTag {
			return nil, "", "", fmt.Errorf("release %s: not found", tagFlag)
		}

		tag = tagFlag

		if err := githubFormatCandidates(&candidates, releaseJson, index); err != nil {
			return nil, "", "", fmt.Errorf("release %s: %s", tag, err.Error())
		}
	}

	return &candidates, pkgName, tag, nil
}

// uses github api to get repo's releases
func githubGetReleases(pkgName string, releaseDepth int) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=%d", pkgName, releaseDepth), nil)
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
func githubGetCandidatesFromRelease(json string, release int64) map[string]string {
	assetCount := gjson.Get(json, fmt.Sprintf("%d.assets.#", release)).Int()
	candidates := make(map[string]string)

	for i := range assetCount {
		assetPath := fmt.Sprintf("%d.assets.%d", release, i)
		candidates[gjson.Get(json, assetPath+".name").String()] = gjson.Get(json, assetPath+".browser_download_url").String()
	}

	return candidates
}

// return to caveman 2
// returns: found, index
func githubTagSearch(json, tag string) (bool, int64) {
	for i := range gjson.Get(json, "#").Int() {
		if tag == gjson.Get(json, fmt.Sprintf("%d.tag_name", i)).String() {
			return true, i
		}
	}

	return false, 0
}

func githubFormatCandidates(candidates *map[string]string, json string, index int64) error {
	// check if any assets are available
	if gjson.Get(json, fmt.Sprintf("%d.assets.#", index)).Int() == 0 {
		return fmt.Errorf("no assets available")
	}

	// get and filter candidates (release files)
	*candidates = githubGetCandidatesFromRelease(json, index)

	if err := filterCandidates(*candidates); err != nil {
		return err
	}

	if len(*candidates) != 1 {
		installUserChoice(*candidates)
	}

	return nil
}
