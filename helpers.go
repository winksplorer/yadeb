package main

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/tidwall/gjson"
)

var architectureAliases = map[string][]string{
	"amd64": {"amd64", "x86_64", "x64"},
	"arm64": {"arm64", "aarch64", "armv8"},
}

func githubGetReleases(user, repo string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", user, repo), nil)
	if err != nil {
		return "", err
	}

	// Set headers like in curl
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read and dump response
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

func filterCandidates(candidates map[string]string) error {
	fmt.Println("filterCandidates: first iteration (*.deb)")
	mapFilter(candidates, func(v string) bool {
		return !strings.HasSuffix(v, ".deb")
	})

	if len(candidates) == 1 {
		return nil
	} else if len(candidates) == 0 {
		return fmt.Errorf("zero candidates remaining, cannot continue")
	}

	fmt.Printf("filterCandidates: second iteration (%s)\n", runtime.GOARCH)

	// match any architecture to see if they exist
	var allArchitectures []string
	for _, v := range architectureAliases {
		allArchitectures = append(allArchitectures, v...)
	}

	archSpecific := false
	for _, v := range candidates {
		if containsAny(v, allArchitectures) {
			archSpecific = true
			break
		}
	}

	if !archSpecific {
		// TODO: ask user which one to download
		fmt.Println("i give up")
		return nil
	}

	// look for current architecture
	mapFilter(candidates, func(v string) bool {
		return !containsAny(v, architectureAliases[runtime.GOARCH])
	})

	if len(candidates) == 1 {
		return nil
	} else if len(candidates) == 0 {
		return fmt.Errorf("zero candidates remaining, cannot continue")
	}

	return nil
}

// filters a string map.
// check should return true if the item should be removed.
func mapFilter(m map[string]string, check func(v string) bool) {
	for k, v := range m {
		if check(v) {
			delete(m, k)
		}
	}
}

// return to caveman
func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}
