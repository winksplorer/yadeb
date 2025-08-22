package main

import (
	"fmt"
	"os"
)

var (
	BuildDate string = "undefined"
	Version   string = "undefined"
)

func main() {
	if len(os.Args) <= 1 {
		helpMenu()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "-v", "--version":
		fmt.Printf("yadeb v%s (built on %s)\n", Version, BuildDate)
	case "help", "-h", "--help":
		helpMenu()
	case "install", "remove", "purge", "upgrade", "upgrade-all", "list", "pin", "selfhost":
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
