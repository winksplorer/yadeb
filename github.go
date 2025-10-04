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
func githubGetCandidates(u *url.URL, tagFlag string, cfg *ini.File) ([]string, string, string, error) {
	// get user and repo
	pkgName, _ := strings.CutPrefix(u.Path, "/")

	var (
		candidates []string
		tag        string
	)

	// go through releases
	if tagFlag == "latest" {
		// get releases
		fmt.Printf("Fetching releases from github.com/%s...", pkgName)
		releaseJson, err := githubGetReleases(pkgName, cfg.Section("yadeb").Key("ReleaseDepth").MustInt(50))
		if err != nil {
			fmt.Println() // Yes, this is bad. Yes, you will see this a lot.
			return nil, "", "", fmt.Errorf("couldn't fetch github releases: %s", err)
		}
		fmt.Println(doneMsg)

		if gjson.Get(releaseJson, "#").Int() == 0 {
			return nil, "", "", fmt.Errorf("requested package has no releases available")
		}

		tag, candidates, err = githubFindLatestValidRelease(releaseJson, cfg)
		if err != nil {
			return nil, "", "", err
		}
	} else {
		fmt.Printf("Fetching github.com/%s at release %s...", pkgName, tagFlag)
		releaseJson, err := githubReleaseByTag(pkgName, tagFlag)
		if err != nil {
			fmt.Println()
			return nil, "", "", fmt.Errorf("release %s: failed to fetch: %s", tagFlag, err.Error())
		}
		fmt.Println(doneMsg)

		if gjson.Get(releaseJson, "status").String() == "404" {
			return nil, "", "", fmt.Errorf("release %s: not found", tagFlag)
		}

		tag = tagFlag

		candidates, err = githubFormatCandidates(releaseJson, "assets")
		if err != nil {
			return nil, "", "", fmt.Errorf("release %s: %s", tag, err.Error())
		}
	}

	return candidates, pkgName, tag, nil
}

func githubApiRequest(link string) (string, error) {
	req, err := http.NewRequest("GET", link, nil)
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

// uses github api to get repo's releases
func githubGetReleases(pkgName string, releaseDepth int) (string, error) {
	return githubApiRequest(fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=%d", pkgName, releaseDepth))
}

// gets a tag
func githubReleaseByTag(pkgName, tag string) (string, error) {
	return githubApiRequest(fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", pkgName, tag))
}

func githubFindLatestValidRelease(json string, cfg *ini.File) (string, []string, error) {
	for i := range gjson.Get(json, "#").Int() {
		// get tag
		tag := gjson.Get(json, fmt.Sprintf("%d.tag_name", i)).String()

		if !cfg.Section("yadeb").Key("AllowPrerelease").MustBool(false) && gjson.Get(json, fmt.Sprintf("%d.prerelease", i)).Bool() {
			fmt.Printf("Skipping release %s: \033[91mrelease is a prerelease, which is disallowed\033[0m\n", tag)
			continue
		}

		candidates, err := githubFormatCandidates(json, fmt.Sprintf("%d.assets", i))
		if err != nil {
			fmt.Printf("Skipping release %s: \033[91m%s\033[0m\n", tag, err.Error())
			continue
		}

		return tag, candidates, nil
	}

	return "", nil, fmt.Errorf("no valid release found")
}

// use githubGetReleases to get the json
func githubGetCandidatesFromRelease(json string, assetsPath string, assetCount int64) []string {
	var candidates []string

	for i := range assetCount {
		assetPath := fmt.Sprintf("%s.%d", assetsPath, i)
		candidates = append(candidates, gjson.Get(json, assetPath+".browser_download_url").String())
	}

	return candidates
}

func githubFormatCandidates(json string, assetsPath string) ([]string, error) {
	assetCount := gjson.Get(json, fmt.Sprintf("%s.#", assetsPath)).Int()

	if assetCount == 0 {
		return nil, fmt.Errorf("no assets available")
	}

	// get and filter candidates (release files)
	candidates := githubGetCandidatesFromRelease(json, assetsPath, assetCount)
	candidates, err := filterCandidates(candidates)

	if err != nil {
		return candidates, err
	}

	return candidates, nil
}
