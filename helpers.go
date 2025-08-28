package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

// return to caveman
func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// \033[91mError:\033[0m {s}
func ansiError(s ...string) {
	fmt.Println("\033[91mError\033[0m:", strings.Join(s, " "))
}

// \n\033[91mError:\033[0m {s}
func lnAnsiError(s ...string) {
	fmt.Println("\n\033[91mError\033[0m:", strings.Join(s, " "))
}

// downloads file
func downloadFile(href, path string) error {
	// check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	} else if !os.IsNotExist(err) {
		return err // ????
	}

	// get the page
	resp, err := http.Get(href)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// create file
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	// download
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// generates random b64 str
func randomBase64(length int) (string, error) {
	numBytes := (length * 3) / 4
	randomBytes := make([]byte, numBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes)[:length], nil
}

// runs apt with args, fully passing stdin, stdout, and stderr
func runApt(args ...string) error {
	cmd := exec.Command("/usr/bin/apt", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if cmd.ProcessState != nil {
			return fmt.Errorf("apt failed with exit code %d", cmd.ProcessState.ExitCode())
		}

		return err
	}

	return nil
}

// chowns a dir/file to _apt:root
func aptChown(path string) error {
	// user lookup
	u, err := user.Lookup("_apt")
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}

	return os.Chown(path, uid, 0)
}

// removes directory
func cleanupDir(path string) error {
	fmt.Print("\nCleaning up...")
	if err := os.RemoveAll(path); err != nil {
		fmt.Println()
		return fmt.Errorf("couldn't delete %s: %s", path, err)
	}
	fmt.Println(doneMsg)

	return nil
}

// creates /tmp/yadeb-16 char b64 string/
func createTempDir() (string, error) {
	b64, err := randomBase64(16)
	if err != nil {
		return "", err
	}

	tempDir := "/tmp/yadeb-" + b64

	if err := os.Mkdir(tempDir, 0755); err != nil {
		return "", err
	}

	if err := aptChown(tempDir); err != nil {
		return "", err
	}

	return tempDir, nil
}

// creates a "unix-style" numbered menu.
// returns: valid, selected index
func numberedMenu(values []string) (bool, int) {
	for i, v := range values {
		fmt.Printf("[%d] %s\n", i+1, v)
	}

	fmt.Print("Enter your option: ")
	var choice int
	_, err := fmt.Scan(&choice)
	if err != nil || choice < 1 || choice > len(values) {
		fmt.Println("Invalid choice")
		return false, 0
	}

	return true, choice - 1
}
