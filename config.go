package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// creates /etc/yadeb
func createConfigDir() error {
	if _, err := os.Stat("/etc/yadeb"); err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir("/etc/yadeb", 0755)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if err := createConfig(); err != nil {
		return fmt.Errorf("createConfig: %s", err.Error())
	}

	return nil
}

// creates and fills in /etc/yadeb/config.ini, IF IT EXISTS
func createConfig() error {
	if _, err := os.Stat("/etc/yadeb/config.ini"); err != nil {
		if os.IsNotExist(err) {
			// ini data
			cfg := ini.Empty()

			sec, err := cfg.NewSection("yadeb")
			if err != nil {
				return err
			}

			if _, err = sec.NewKey("Version", Version); err != nil {
				return err
			}

			if _, err = sec.NewKey("AllowPrerelease", "false"); err != nil {
				return err
			}

			if _, err = sec.NewKey("ReleaseDepth", "50"); err != nil {
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

// marks a package as installed in /etc/yadeb/installed.ini, creating it if necessary
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

	// I LOVE GO!!!
	// do i REALLY need a /s there?

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

// unmarks a package as installed
func unmarkAsInstalled(link string) error {
	// file not exist logic
	if _, err := os.Stat("/etc/yadeb/installed.ini"); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}

	// load
	cfg, err := ini.Load("/etc/yadeb/installed.ini")
	if err != nil {
		return err
	}

	// remove
	cfg.DeleteSection(link)

	// save
	if err = cfg.SaveTo("/etc/yadeb/installed.ini"); err != nil {
		return err
	}

	return nil
}

// updates a package's install mark
func updatePackageMark(link, tag string) error {
	// file not exist logic
	if _, err := os.Stat("/etc/yadeb/installed.ini"); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("install database doesn't exist")
		} else {
			return err
		}
	}

	// load
	cfg, err := ini.Load("/etc/yadeb/installed.ini")
	if err != nil {
		return err
	}

	// update
	for _, sec := range cfg.Sections() {
		if sec.Name() == link {
			sec.Key("InstalledTag").SetValue(tag)
			sec.Key("LastUpdate").SetValue(time.Now().Format("2006-01-02"))
		}
	}

	// save
	if err = cfg.SaveTo("/etc/yadeb/installed.ini"); err != nil {
		return err
	}

	return nil
}

// gets a tracked package by link
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
			if err = section.MapTo(&p); err != nil {
				return nil, err
			}

			return &p, nil
		}
	}

	return nil, nil
}

// gets all tracked packages
func getAllPackages() ([]Package, error) {
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

	var pkgs []Package

	for _, section := range cfg.Sections() {
		var p Package
		p.Link = section.Name()
		if err = section.MapTo(&p); err != nil {
			return nil, err
		}

		pkgs = append(pkgs, p)
	}

	return pkgs, nil
}
