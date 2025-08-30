package main

import (
	"fmt"
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

	// "normalize" the url (this might still cause issues)
	raw := links[0]
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	p, err := getPackage(raw)
	if err != nil {
		ansiError("Couldn't read installed package database:", err.Error())
		return 1
	}

	if p == nil || p.Package == "" {
		fmt.Println(p)
		ansiError("Requested package isn't installed")
		return 1
	}

	// actually uninstall
	fmt.Print("Starting APT...\n\n")
	if err := runApt("remove", p.Package); err != nil {
		ansiError("Couldn't run apt:", err.Error())
		return 1
	}

	if err := unmarkAsInstalled(raw); err != nil {
		ansiError("Couldn't remove installation mark:", err.Error())
		return 1
	}

	return 0
}
