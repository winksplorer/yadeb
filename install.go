package main

import (
	"fmt"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"

	"gopkg.in/ini.v1"
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

	cfg, err := ini.Load("/etc/yadeb/config.ini")
	if err != nil {
		ansiError("Couldn't read /etc/yadeb/config.ini")
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

	var (
		candidates *map[string]string
		pkgName    string
		tag        string
	)

	// decide what to do based on domain
	switch u.Host {
	case "github.com":
		candidates, pkgName, tag, err = githubGetCandidates(u, tagFlag, cfg)
	default:
		ansiError("Unknown source domain:", u.Host)
		return 2
	}

	if err != nil {
		ansiError("Failed to get candidates:", err.Error())
		return 1
	}

	// downlad the remaining candidate
	for _, v := range *candidates {
		if err := candidateInstall(pkgName, tag, v, u); err != nil {
			ansiError(fmt.Sprintf("Couldn't install %s: %s", pkgName, err.Error()))
			return 1
		}

		return 0
	}

	ansiError("No candidate to install, somehow")
	return 1
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
		installUserChoice(candidates)
		return nil
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
func candidateInstall(pkgName, tag, downloadLink string, u *url.URL) error {
	// create
	tempDir, err := createTempDir()
	if err != nil {
		return fmt.Errorf("couldn't create temp directory: %s", err)
	}

	path := fmt.Sprintf("%s/%s", tempDir, filepath.Base(downloadLink))

	// download
	fmt.Printf("Downloading %s from release %s...", filepath.Base(downloadLink), tag)
	if err := downloadFile(downloadLink, path); err != nil {
		fmt.Println()
		cleanupDir(tempDir) // Yes, I want to use a defer, but I need to return the value at the end so I can't.
		return fmt.Errorf("couldn't download selected candidate: %s", err)
	}
	fmt.Println(doneMsg)

	// mark
	fmt.Printf("Marking %s as installed...", pkgName)
	if err := markAsInstalled(path, u.String(), tag); err != nil {
		fmt.Println()
		cleanupDir(tempDir)
		return fmt.Errorf("couldn't mark %s as installed: %s", pkgName, err)
	}
	fmt.Println(doneMsg)

	// apt
	fmt.Print("Starting APT...\n\n")
	if err := runApt("install", path); err != nil {
		// if apt fails then unmark the package
		fmt.Printf("Removing installation mark for %s...", pkgName)
		if err := unmarkAsInstalled(u.String()); err != nil {
			fmt.Println()
			cleanupDir(tempDir)
			ansiError("couldn't run apt:", err.Error())
			return fmt.Errorf("couldn't remove installation mark for %s: %s", pkgName, err)
		}
		fmt.Println(doneMsg)

		return fmt.Errorf("couldn't run apt: %s", err)
	}

	return cleanupDir(tempDir)
}

// asks user which remaining candidate to install
func installUserChoice(candidates map[string]string) {
	fmt.Println("There are multiple package files that can be installed. Choose which one to install:")

	candidateValues := slices.Collect(maps.Values(candidates))
	valid := false
	index := 0

	slices.Sort(candidateValues)

	for !valid {
		valid, index = numberedMenu(candidateValues)
	}

	mapFilter(candidates, func(v string) bool {
		return v != candidateValues[index]
	})
}
