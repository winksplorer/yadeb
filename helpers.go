package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tidwall/gjson"
)

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

// downloads file
func downloadFile(href, path string) error {
	// check if file already exists
	if _, err := os.Stat(path); err == nil {
		fmt.Println("already downloaded", path)
		return nil
	} else if !os.IsNotExist(err) {
		return err // ????
	}

	// get the page
	resp, err := http.Get(href)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// create file
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	// download
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("downloaded %s to %s\n", href, path)
	return nil
}

// generates random b64 str
func randomBase64(length int) (string, error) {
	numBytes := (length * 3) / 4
	randomBytes := make([]byte, numBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes)[:length], nil
}
