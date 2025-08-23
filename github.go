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
		fmt.Println("yadeb: invalid github repo link (not enough path parts)")
		return 2
	}

	user := pathParts[1]
	repo := pathParts[2]

	if user == "" || repo == "" {
		fmt.Println("yadeb: invalid github repo link (empty user or repo)")
		return 2
	}

	// get releases
	releaseJson, err := githubGetReleases(user, repo)
	if err != nil {
		fmt.Println("yadeb: couldn't get github releases:", err.Error())
		return 1
	}

	// if no candidates in latest release then error out
	// TODO! CHECK THE OTHER RELEASES!!!!
	if gjson.Get(releaseJson, "0.assets.#").Int() == 0 {
		fmt.Println("yadeb: requested package has no releases available")
		return 1
	}

	// get and filter candidates (release files)
	candidates, _ := githubGetCandidates(releaseJson)

	if err := filterCandidates(candidates); err != nil {
		fmt.Println("yadeb:", err.Error())
		return 1
	}

	if len(candidates) != 1 {
		fmt.Println("yadeb: too many candidates (TODO: let user choose)")
		return 1
	}

	// generate tmp dir
	b64, err := randomBase64(16)
	if err != nil {
		fmt.Println("yadeb: couldn't generate tmp id:", err.Error())
		return 1
	}

	tempDir := "/tmp/yadeb-" + b64

	if err := os.Mkdir(tempDir, 0600); err != nil {
		fmt.Println("yadeb: couldn't generate tmp folder:", err.Error())
		return 1
	}

	// bad but it's fine (for now). downlad the remaining candidate
	for _, v := range candidates {
		if err := downloadFile(v, fmt.Sprintf("%s/%s", tempDir, v[strings.LastIndex(v, "/")+1:])); err != nil {
			fmt.Println("yadeb: couldn't download selected candidate:", err.Error())
			return 1
		}

		break
	}

	return 0
}

// uses github api to get repo's releases
func githubGetReleases(user, repo string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=100", user, repo), nil)
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
