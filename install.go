package main

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"slices"
	"strings"
	"syscall"

	"github.com/tidwall/gjson"
)

func cmdInstall() int {
	if len(os.Args) <= 2 {
		fmt.Println("yadeb: nothing to install")
		return 2
	}

	if syscall.Geteuid() != 0 {
		fmt.Println("yadeb: installation requires root privileges")
		os.Exit(2)
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

		releaseJson, err := githubGetReleases(user, repo)
		if err != nil {
			fmt.Println("yadeb: couldn't get github releases:", err.Error())
			return 1
		}

		if gjson.Get(releaseJson, "0.assets.#").Int() == 0 {
			fmt.Println("yadeb: requested package has no releases available")
			return 1
		}

		candidates, _ := githubGetCandidates(releaseJson)

		if err := filterCandidates(candidates); err != nil {
			fmt.Println("yadeb:", err.Error())
			return 1
		}

		b64, err := randomBase64(16)
		if err != nil {
			fmt.Println("yadeb: couldn't generate tmp id:", err.Error())
			return 1
		}

		tempDir := "/tmp/yadeb-" + b64

		if err := os.Mkdir(tempDir, 0666); err != nil {
			fmt.Println("yadeb: couldn't generate tmp folder:", err.Error())
			return 1
		}

		for _, v := range candidates {
			if err := downloadFile(v, fmt.Sprintf("%s/%s", tempDir, v[strings.LastIndex(v, "/")+1:])); err != nil {
				fmt.Println("yadeb: couldn't download selected candidate:", err.Error())
				return 1
			}

			break
		}

	default:
		fmt.Printf("yadeb: unknown source domain (%s)\n", u.Host)
		return 2
	}

	return 0
}

func filterCandidates(candidates map[string]string) error {
	fmt.Println("first iteration (*.deb)")
	mapFilter(candidates, func(v string) bool {
		return !strings.HasSuffix(v, ".deb")
	})

	if len(candidates) == 1 {
		return nil
	} else if len(candidates) == 0 {
		return fmt.Errorf("zero candidates remaining, cannot continue")
	}

	fmt.Printf("second iteration (%s)\n", runtime.GOARCH)

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
