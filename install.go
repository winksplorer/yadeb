package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
)

// the install command
func cmdInstall(links []string, tagFlag string) int {
	if len(links) == 0 {
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
	raw := links[0]
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

	// init architecture slice
	for _, v := range architectureAliases {
		allArchitectures = append(allArchitectures, v...)
	}

	// decide what to do based on domain
	switch u.Host {
	case "github.com":
		return githubCmdInstall(u, tagFlag)
	default:
		ansiError("Unknown source domain:", u.Host)
		return 2
	}
}

// filters candidates from name
func filterCandidates(candidates map[string]string) error {
	// .deb filtering
	mapFilter(candidates, func(v string) bool {
		return !strings.HasSuffix(v, ".deb")
	})

	if len(candidates) == 1 {
		return nil
	} else if len(candidates) == 0 {
		return fmt.Errorf("no package files found")
	}

	// match any arch to see if they exist
	archSpecific := false
	for _, v := range candidates {
		if containsAny(v, allArchitectures) {
			archSpecific = true
			break
		}
	}

	if !archSpecific {
		fmt.Println() // WOW this is shit
		return fmt.Errorf("multiple package files yet no architecture information (TODO: let user choose)")
	}

	// look for current architecture
	mapFilter(candidates, func(v string) bool {
		return !containsAny(v, architectureAliases[runtime.GOARCH])
	})

	if len(candidates) == 0 {
		return fmt.Errorf("no package files for %s", runtime.GOARCH)
	}

	return nil
}

// installs a candidate
func candidateInstall(user, repo, tempDir, tag, downloadLink string, u *url.URL) int {
	path := fmt.Sprintf("%s/%s", tempDir, filepath.Base(downloadLink))

	fmt.Printf("Downloading %s from release %s...", filepath.Base(downloadLink), tag)
	if err := downloadFile(downloadLink, path); err != nil {
		lnAnsiError("Couldn't download selected candidate:", err.Error())
		cleanupDir(tempDir)
		return 1
	}
	fmt.Println(doneMsg)

	fmt.Printf("Marking %s/%s as installed...", user, repo)
	if err := markAsInstalled(path, u.String(), tag); err != nil {
		lnAnsiError(fmt.Sprintf("Couldn't mark %s/%s as installed:", user, repo), err.Error())
		cleanupDir(tempDir)
		return 1
	}
	fmt.Println(doneMsg)

	fmt.Print("Starting APT...\n\n")
	if err := runApt("install", path); err != nil {
		ansiError("Couldn't run APT:", err.Error())

		// if apt fails then unmark it
		fmt.Printf("Removing installation mark for %s/%s...", user, repo)
		if err := unmarkAsInstalled(u.String()); err != nil {
			lnAnsiError(fmt.Sprintf("Couldn't remove installation mark for %s/%s:", user, repo))
			cleanupDir(tempDir)
			return 1
		}
		fmt.Println(doneMsg)
	}

	return cleanupDir(tempDir)
}
