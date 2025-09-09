package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/tidwall/gjson"
	"gopkg.in/ini.v1"
)

// the upgrade command
func cmdUpgrade(links []string) int {
	if len(links) == 0 {
		ansiError("Nothing to upgrade")
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

	// get existing package
	p, err := getPackage(u.String())
	if err != nil {
		ansiError("Couldn't read installed package database:", err.Error())
		return 1
	}
	if p == nil {
		ansiError("Requested package isn't installed")
		return 1
	}

	// init architecture slice
	for _, v := range architectureAliases {
		allArchitectures = append(allArchitectures, v...)
	}

	var (
		candidates []string
		pkgName    string
		tag        string
	)

	// decide what to do based on domain
	switch u.Host {
	case "github.com":
		// get user and repo
		pkgName, _ := strings.CutPrefix(u.Path, "/")

		// get releases
		fmt.Printf("Asking GitHub for releases on %s...", pkgName)
		releaseJson, err := githubGetReleases(pkgName, cfg.Section("yadeb").Key("ReleaseDepth").MustInt(50))
		if err != nil {
			lnAnsiError("couldn't get github releases:", err.Error())
			return 1
		}
		fmt.Println(doneMsg)

		if gjson.Get(releaseJson, "#").Int() == 0 {
			lnAnsiError("requested package has no releases available")
			return 1
		}

		tag, candidates, err = githubFindLatestRelease(releaseJson, cfg)
		if err != nil {
			lnAnsiError(err.Error())
			return 1
		}
	default:
		ansiError("Unknown source domain:", u.Host)
		return 2
	}

	if p.InstalledTag == tag {
		fmt.Fprintln(os.Stderr, u.String(), "is already at the latest version")
		return 0
	}

	// downlad the remaining candidate
	if err := candidateUpgrade(pkgName, tag, candidates[0], u); err != nil {
		ansiError(fmt.Sprintf("Couldn't upgrade %s: %s", pkgName, err.Error()))
		return 1
	}

	return 0
}

// upgrades a candidate
func candidateUpgrade(pkgName, tag, downloadLink string, u *url.URL) error {
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

	// apt
	fmt.Print("Starting APT (install)...\n\n")
	if err := runApt("install", path); err != nil {
		return fmt.Errorf("couldn't run apt: %s", err)
	}

	// mark
	fmt.Printf("Marking %s as updated...", pkgName)
	if err := updatePackageMark(u.String(), tag); err != nil {
		fmt.Println()
		cleanupDir(tempDir)
		return fmt.Errorf("couldn't mark %s as updated: %s", pkgName, err)
	}
	fmt.Println(doneMsg)

	return cleanupDir(tempDir)
}
