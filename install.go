package main

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"slices"
	"strings"
	"syscall"
)

func cmdInstall() int {
	if len(os.Args) <= 2 {
		fmt.Println("yadeb: nothing to install")
		return 2
	}

	if syscall.Geteuid() != 0 {
		fmt.Println("yadeb: installation requires root privileges")
		return 2
	}

	if err := createConfigDir(); err != nil {
		fmt.Println("yadeb: couldn't create (or check existence of) /etc/yadeb")
		return 1
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
		return githubCmdInstall(u)
	default:
		fmt.Printf("yadeb: unknown source domain (%s)\n", u.Host)
		return 2
	}
}

func filterCandidates(candidates map[string]string) error {
	// .deb filtering
	fmt.Print("First candidate iteration (*.deb)...")
	mapFilter(candidates, func(v string) bool {
		return !strings.HasSuffix(v, ".deb")
	})
	fmt.Println(doneMsg)

	if len(candidates) == 1 {
		return nil
	} else if len(candidates) == 0 {
		return fmt.Errorf("zero candidates remaining, cannot continue")
	}

	// arch filtering
	fmt.Printf("Second candidate iteration (%s)...", runtime.GOARCH)

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
		return fmt.Errorf("multiple candidates remaining yet no architecture information, cannot continue (TODO: let user choose)")
	}

	// look for current architecture
	mapFilter(candidates, func(v string) bool {
		return !containsAny(v, architectureAliases[runtime.GOARCH])
	})

	fmt.Println(doneMsg)

	if len(candidates) == 0 {
		return fmt.Errorf("zero candidates remaining, cannot continue")
	}

	return nil
}
