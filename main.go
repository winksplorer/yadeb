package main

import (
	"fmt"
	"os"
	"time"
)

var (
	BuildDate string = "undefined"
	Version   string = "undefined"

	// supported architectures (GOARCH format) and aliases to all the weird names people give them
	architectureAliases = map[string][]string{
		"386":     {"i386", "i686", "ia32", "x86"},
		"amd64":   {"amd64", "x86_64", "x86-64", "x64"},
		"arm":     {"armhf", "armel", "armv7"}, // TODO: this is bad. armhf != armel. this WILL cause problems later. TODO!!!!
		"arm64":   {"arm64", "aarch64", "armv8"},
		"ppc64le": {"ppc64le", "ppc64el"},
		"riscv64": {"riscv64", "rv64", "risc-v64"},
		"s390x":   {"s390x"},
	}
)

const (
	// green "done" string
	doneMsg string = " \033[92mDone\033[0m"
)

type (
	// a tracked "package"
	Package struct {
		Name         string
		Link         string
		InstalledTag string
		InstallDate  time.Time
		LastUpdate   time.Time
	}
)

// entry point
func main() {
	if len(os.Args) <= 1 {
		helpMenu()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "-v", "--version":
		fmt.Printf("yadeb v%s (built on %s)\n", Version, BuildDate)
	case "install":
		os.Exit(cmdInstall())
	case "remove", "purge", "upgrade", "upgrade-all", "list", "pin", "selfhost":
		fmt.Println("not implemented")
		os.Exit(2)
	default:
		helpMenu()
		os.Exit(2)
	}
}

// shows help message
func helpMenu() {
	// TODO: maybe use a different word instead of packages?
	fmt.Printf(
		"yadeb v%s (built on %s)\n"+
			"Usage: %s command [options] [links]\n\n"+
			"All commands:\n"+
			"  install - installs packages\n"+
			"  remove - removes packages\n"+
			"  purge - purges packages\n"+
			"  upgrade - upgrades packages\n"+
			"  upgrade-all - upgrades all installed packages\n"+
			"  list - lists installed packages\n"+
			"  pin - pins a package to a specific version\n"+
			"  selfhost - reinstalls yadeb itself as a package\n",

		Version, BuildDate, os.Args[0],
	)
}
