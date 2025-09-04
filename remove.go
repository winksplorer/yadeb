package main

import (
	"fmt"
	"net/url"
	"strings"
	"syscall"
)

func cmdRemove(links []string) int {
	if len(links) == 0 {
		ansiError("Nothing to remove")
		return 2
	}

	if syscall.Geteuid() != 0 {
		ansiError("Removing a package requires root privileges")
		return 2
	}

	// "normalize" the url
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

	// get user and repo
	pkgName, _ := strings.CutPrefix(u.Path, "/")

	fmt.Printf("Checking if %s is installed...", pkgName)
	p, err := getPackage(raw)
	if err != nil {
		lnAnsiError("Couldn't read installed package database:", err.Error())
		return 1
	}

	if p == nil || p.Package == "" {
		fmt.Println(p)
		ansiError("Requested package isn't installed")
		return 1
	}
	fmt.Println(doneMsg)

	// actually uninstall
	fmt.Print("Starting APT...\n\n")
	if err := runApt("remove", p.Package); err != nil {
		ansiError("Couldn't run apt:", err.Error())
		return 1
	}

	fmt.Printf("\n\nRemoving installation mark from %s...", pkgName)
	if err := unmarkAsInstalled(raw); err != nil {
		ansiError("Couldn't remove installation mark:", err.Error())
		return 1
	}
	fmt.Println(doneMsg)

	return 0
}
