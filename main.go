package main

import (
	"flag"
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

	// all architecture aliases in a single slice
	allArchitectures []string
)

const (
	// green "done" string
	doneMsg string = " \033[92mDone\033[0m"
)

type (
	// a tracked "package"
	Package struct {
		Package      string
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

	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s %s [options] [links]\n\n", os.Args[0], os.Args[1])
		fmt.Fprintf(os.Stderr, "All options:\n")
		fs.PrintDefaults()
	}

	switch os.Args[1] {
	case "-v", "--version":
		fmt.Printf("yadeb v%s (built on %s)\n", Version, BuildDate)
	case "install":
		tagFlag := fs.String("tag", "latest", "Release/GitHub tag")

		fs.Parse(os.Args[2:])
		os.Exit(cmdInstall(fs.Args(), *tagFlag))
	case "remove", "purge":
		fs.Parse(os.Args[2:])
		os.Exit(cmdRemove(fs.Args()))
	case "upgrade":
		fs.Parse(os.Args[2:])
		os.Exit(cmdUpgrade(fs.Args()))
	case "upgrade-all", "list":
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
			"For more info about a command, type '%s <command> --help'.\n",

		Version, BuildDate, os.Args[0], os.Args[0],
	)
}
