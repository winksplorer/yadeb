package main

import (
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/tidwall/gjson"
)

func cmdInstall() int {
	if len(os.Args) <= 2 {
		fmt.Println("yadeb: nothing to install")
		return 2
	}

	// "common hack"
	raw := os.Args[2]
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	// parse link
	u, err := url.Parse(raw)
	if err != nil {
		fmt.Println("yadeb: couldn't parse link:", err.Error())
		return 1
	}

	// error out if unknown scheme
	if !slices.Contains([]string{"http", "https", ""}, u.Scheme) {
		fmt.Printf("yadeb: unknown source scheme (%s)\n", u.Scheme)
		return 2
	}

	// decide what to do based on domain
	switch u.Host {
	case "github.com":
		fmt.Println("it is a github link")
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

		fmt.Printf("user: %s\nrepo: %s\n", user, repo)

		releaseJson, err := githubGetReleases(user, repo)
		if err != nil {
			fmt.Println("yadeb: couldn't get github releases:", err.Error())
			return 1
		}

		fmt.Println(gjson.Get(releaseJson, "0.assets.#"))

		a, _ := githubGetCandidates(releaseJson)

		for key, val := range a {
			fmt.Printf("Key: %s, Value: %s\n", key, val)
		}

	default:
		fmt.Printf("yadeb: unknown source domain (%s)\n", u.Host)
		return 2
	}

	return 0
}
