package main

import (
	"fmt"
	"net/url"
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
		ansiError("Upgrading requires root privileges")
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
	if !slices.Contains([]string{"https", ""}, u.Scheme) {
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
		fmt.Printf("Checking github.com/%s...", pkgName)
		releaseJson, err := githubGetReleases(pkgName, cfg.Section("yadeb").Key("ReleaseDepth").MustInt(50))
		if err != nil {
			lnAnsiError("couldn't get github releases:", err.Error())
			return 1
		}

		if gjson.Get(releaseJson, "#").Int() == 0 {
			lnAnsiError("requested package has no releases available")
			return 1
		}

		tag, candidates, err = githubFindLatestValidRelease(releaseJson, cfg)
		if err != nil {
			lnAnsiError(err.Error())
			return 1
		}
	default:
		ansiError("Unknown source domain:", u.Host)
		return 2
	}

	if p.InstalledTag == tag {
		fmt.Printf(" \033[92mAlready at latest (%s)\033[0m\n", tag)
		return 0
	}

	fmt.Printf(" \033[92mNew version available (%s)\033[0m\n", tag)

	if len(candidates) != 1 {
		installUserChoice(candidates)
	}

	pii := PackageToInstall{
		Name:         pkgName,
		Tag:          tag,
		DownloadLink: candidates[0],
		Url:          u,
	}

	if len(candidates) != 1 {
		installUserChoice(candidates)
	}

	// downlad the remaining candidate
	if err := candidateUpgrade(pii); err != nil {
		ansiError(fmt.Sprintf("Couldn't upgrade %s: %s", pkgName, err.Error()))
		return 1
	}

	return 0
}

// the upgrade command
func cmdUpgradeAll() int {
	if syscall.Geteuid() != 0 {
		ansiError("Upgrading requires root privileges")
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

	// init architecture slice
	for _, v := range architectureAliases {
		allArchitectures = append(allArchitectures, v...)
	}

	var pii []PackageToInstall

	pkgs, err := getAllPackages()
	if err != nil {
		ansiError(err.Error())
	}

	if pkgs == nil {
		return 0
	}

	for _, p := range pkgs {
		if p.Link == "DEFAULT" {
			continue
		}

		// parse link
		u, err := url.Parse(p.Link)
		if err != nil {
			ansiError("Couldn't parse link:", err.Error())
			return 1
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
			pkgName, _ = strings.CutPrefix(u.Path, "/")

			// get releases
			fmt.Printf("Checking github.com/%s...", pkgName)
			releaseJson, err := githubGetReleases(pkgName, cfg.Section("yadeb").Key("ReleaseDepth").MustInt(50))
			if err != nil {
				lnAnsiError("couldn't get github releases:", err.Error())
				return 1
			}

			if gjson.Get(releaseJson, "#").Int() == 0 {
				lnAnsiError("requested package has no releases available")
				continue
			}

			tag, candidates, err = githubFindLatestValidRelease(releaseJson, cfg)
			if err != nil {
				lnAnsiError(err.Error())
				return 1
			}
		default:
			ansiError("Unknown source domain:", u.Host)
			return 2
		}

		if p.InstalledTag == tag {
			fmt.Printf(" \033[92mAlready at latest (%s)\033[0m\n", tag)
			continue
		}

		fmt.Printf(" \033[92mNew version available (%s)\033[0m\n", tag)

		if len(candidates) != 1 {
			installUserChoice(candidates)
		}

		pii = append(pii, PackageToInstall{
			Name:         pkgName,
			Tag:          tag,
			DownloadLink: candidates[0],
			Url:          u,
		})
	}

	if len(pii) == 0 {
		return 0
	}

	// downlad the remaining candidate
	if err := candidateUpgrade(pii...); err != nil {
		ansiError(fmt.Sprintf("Couldn't upgrade everything: %s", err.Error()))
		return 1
	}

	return 0
}

// upgrades candidates
func candidateUpgrade(pkgs ...PackageToInstall) error {
	// create
	tempDir, err := createTempDir()
	if err != nil {
		return fmt.Errorf("couldn't create temp directory: %s", err)
	}

	for _, p := range pkgs {
		path := fmt.Sprintf("%s/%s", tempDir, filepath.Base(p.DownloadLink))

		// download
		fmt.Printf("Downloading %s from %s at tag %s...", filepath.Base(p.DownloadLink), p.Name, p.Tag)
		if err := downloadFile(p.DownloadLink, path); err != nil {
			lnAnsiError(fmt.Sprintf("Couldn't download %s:", p.Name), err.Error())
			continue
		}
		fmt.Println(doneMsg)

		// apt
		fmt.Print("Starting APT (install)...\n\n")
		if err := runApt("install", "-y", path); err != nil {
			ansiError("Couldn't run apt: %s", err.Error())
			continue
		}

		// mark
		fmt.Printf("Marking %s as updated...", p.Name)
		if err := updatePackageMark(p.Url.String(), p.Tag); err != nil {
			ansiError(fmt.Sprintf("Couldn't mark %s as updated:", p.Name), err.Error())
		}
		fmt.Println(doneMsg)
	}

	return cleanupDir(tempDir)
}
