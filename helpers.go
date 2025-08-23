package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"gopkg.in/ini.v1"
)

// filters a string map.
// check should return true if the item should be removed.
func mapFilter(m map[string]string, check func(v string) bool) {
	for k, v := range m {
		if check(v) {
			delete(m, k)
		}
	}
}

// return to caveman
func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// downloads file
func downloadFile(href, path string) error {
	// check if file already exists
	if _, err := os.Stat(path); err == nil {
		fmt.Println("already downloaded", path)
		return nil
	} else if !os.IsNotExist(err) {
		return err // ????
	}

	// get the page
	resp, err := http.Get(href)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

	fmt.Printf("downloaded %s to %s\n", href, path)
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

func createConfigDir() error {
	if _, err := os.Stat("/etc/yadeb"); err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir("/etc/yadeb", 0644)
			if err != nil {
				return err
			}

			if err = createConfig(); err != nil {
				return fmt.Errorf("createConfig: %s", err.Error())
			}
		} else {
			return err
		}
	}

	return nil
}

func createConfig() error {
	if _, err := os.Stat("/etc/yadeb/config.ini"); err != nil {
		if os.IsNotExist(err) {
			cfg := ini.Empty()

			sec, err := cfg.NewSection("yadeb")
			if err != nil {
				return err
			}

			if _, err = sec.NewKey("version", Version); err != nil {
				return err
			}

			if _, err = sec.NewKey("allowPrerelease", "false"); err != nil {
				return err
			}

			if err = cfg.SaveTo("/etc/yadeb/config.ini"); err != nil {
				return err
			}

			if err = os.Chmod("/etc/yadeb/config.ini", 0644); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
