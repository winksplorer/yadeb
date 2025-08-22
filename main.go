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
	}

	switch os.Args[1] {
	case "-v", "--version":
		fmt.Printf("yadeb v%s (built on %s)\n", Version, BuildDate)
	}
}

func helpMenu() {
	fmt.Printf("yadeb v%s (built on %s)\nUsage: %s command [options] [link]\n", Version, BuildDate, os.Args[0])
	os.Exit(2)
}
