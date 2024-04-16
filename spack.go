package spack

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Spack struct {
	exe  string
	args []string
}

func New(exe string, args ...string) *Spack {
	return &Spack{
		exe:  exe,
		args: args,
	}
}

func (s *Spack) exec(args ...string) *exec.Cmd {
	cmd := exec.Command(s.exe, append(append([]string{}, s.args...), args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}

type config struct {
	Config struct {
		InstallTree struct {
			Root string `yaml:"root"`
		} `yaml:"install_tree"`
	} `yaml:"config"`
}

func (s *Spack) GetInstallRoot() (string, error) {
	pr, pw := io.Pipe()
	cmd := s.exec("config", "get", "config")
	cmd.Stdout = pw

	go func() {
		cmd.Run()
		pw.Close()
	}()

	var c config

	err := yaml.NewDecoder(pr).Decode(&c)
	if err != nil {
		return "", err
	}

	return c.Config.InstallTree.Root, nil
}

type Package struct {
	Name    string `json:"name"`
	Version string `json:"latest_version"`
}

func (s *Spack) ListLatestPackages() ([]Package, error) {
	cmd := s.exec("list", "--format version_json")
	pr, pw := io.Pipe()

	cmd.Stdout = pw

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var packages []Package

	if err := json.NewDecoder(pr).Decode(&packages); err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return packages, nil
}

type SpackDB struct {
	Database struct {
		Installs map[string]*Install `json:"installs"`
	} `json:"database"`
}

type Install struct {
	Spec struct {
		Name         string `json:"name"`
		Version      string `json:"version"`
		Hash         string `json:"hash"`
		Dependencies []struct {
			Name   string `json:"name"`
			Hash   string `json:"hash"`
			Params struct {
				DepTypes []string `json:"deptypes"`
			} `json:"parameters"`
		} `json:"dependencies"`
	} `json:"spec"`
	Path        string  `json:"path"`
	InstallTime float64 `json:"installation_time"`
}

func versionCompare(new, old string) int {
	a := strings.Split(new, ".")
	b := strings.Split(old, ".")

	for n, pa := range a {
		if n >= len(b) {
			return 1
		}

		pb := b[n]

		fa, erra := strconv.ParseUint(pa, 10, 64)
		fb, errb := strconv.ParseUint(pb, 10, 64)

		if erra != nil {
			if errb != nil {
				if pa != pb {
					if pa > pb {
						return 1
					}

					return -1
				}
			}

			return 1
		} else if errb != nil {
			return -1
		}

		if fa > fb {
			return 1
		} else if fa < fb {
			return -1
		}
	}

	return 0
}

func (i *Install) NewerThan(j *Install) bool {
	switch versionCompare(i.Spec.Version, j.Spec.Version) {
	case -1:
		return false
	case 0:
		if j.InstallTime > i.InstallTime {
			return false
		}
	}

	return true
}

func (s *Spack) GetInstalledPackages() (map[string]*Install, error) {
	root, err := s.GetInstallRoot()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filepath.Join(root, ".spack-db", "index.json"))
	if err != nil {
		return nil, err
	}

	defer f.Close()

	var db SpackDB

	if err := json.NewDecoder(f).Decode(&db); err != nil {
		return nil, err
	}

	return db.Database.Installs, nil
}

func (s *Spack) Install(pkg string, extra ...string) error {
	return s.exec(append([]string{"install", "-U", "--deprecated", "--fail-fast", pkg}, extra...)...).Run()
}

func (s *Spack) CleanupBuilds() error {
	return s.exec("clean", "-s").Run()
}
