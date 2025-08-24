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

// the install command
func cmdInstall() int {
	if len(os.Args) <= 2 {
		ansiError("Nothing to install")
		return 2
	}

	if syscall.Geteuid() != 0 {
		ansiError("Installation requires root privileges")
		return 2
	}

	if err := createConfigDir(); err != nil {
		ansiError("Couldn't create (or check existence of) /etc/yadeb")
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
		ansiError("Couldn't parse link:", err.Error())
		return 1
	}

	// error out if unknown scheme
	if !slices.Contains([]string{"http", "https", ""}, u.Scheme) {
		ansiError("Unknown source scheme:", u.Scheme)
		return 2
	}

	// error if already installed
	p, err := getPackage(u.String())
	if err != nil {
		ansiError("Couldn't read installed package database:", err.Error())
		return 1
	}
	if p != nil {
		fmt.Fprintln(os.Stderr, u.String(), "is already installed")
		return 0
	}

	// decide what to do based on domain
	switch u.Host {
	case "github.com":
		return githubCmdInstall(u)
	default:
		ansiError("Unknown source domain:", u.Host)
		return 2
	}
}

// filters candidates from name
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
		fmt.Println() // WOW this is shit
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
