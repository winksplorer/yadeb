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
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	fmt.Printf("Downloading %s...", filepath.Base(href))

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

	fmt.Println(doneMsg)
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
			// ini data
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

			// save ini file
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

func markAsInstalled(debFile, link, installedTag string) error {
	out, err := exec.Command("dpkg-deb", "--field", debFile, "Package").Output()
	if err != nil {
		return err
	}
	pkg := strings.TrimSpace(string(out))

	// get base ini data
	var cfg *ini.File
	if _, err := os.Stat("/etc/yadeb/installed.ini"); err != nil {
		if os.IsNotExist(err) {
			cfg = ini.Empty()
		} else {
			return err
		}
	} else {
		cfg, err = ini.Load("/etc/yadeb/installed.ini")
		if err != nil {
			return err
		}
	}

	// data
	sec, err := cfg.NewSection(link)
	if err != nil {
		return err
	}

	if _, err = sec.NewKey("Package", pkg); err != nil {
		return err
	}

	if _, err = sec.NewKey("InstalledTag", installedTag); err != nil {
		return err
	}

	if _, err = sec.NewKey("InstallDate", time.Now().Format("2006-01-02")); err != nil {
		return err
	}

	if _, err = sec.NewKey("LastUpdate", time.Now().Format("2006-01-02")); err != nil {
		return err
	}

	// save ini file
	if err = cfg.SaveTo("/etc/yadeb/installed.ini"); err != nil {
		return err
	}

	if err = os.Chmod("/etc/yadeb/installed.ini", 0644); err != nil {
		return err
	}

	return nil
}

func getPackage(link string) (*Package, error) {
	if _, err := os.Stat("/etc/yadeb/installed.ini"); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	cfg, err := ini.Load("/etc/yadeb/installed.ini")
	if err != nil {
		return nil, err
	}

	for _, section := range cfg.Sections() {
		if section.Name() == link {
			var p Package
			p.Link = section.Name()
			section.MapTo(&p)
			return &p, nil
		}
	}

	return nil, nil
}

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
