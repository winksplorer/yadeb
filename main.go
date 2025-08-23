package main

import (
	"fmt"
	"os"
)

var (
	BuildDate           string = "undefined"
	Version             string = "undefined"
	architectureAliases        = map[string][]string{
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
	doneMsg string = " \033[92mDone\033[0m"
)

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
