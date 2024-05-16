package spack

import (
	"bytes"
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
	exe    string
	args   []string
	config config
}

func New(exe string, args ...string) (*Spack, error) {
	s := &Spack{
		exe:  exe,
		args: args,
	}

	if err := s.readConfig(); err != nil {
		return nil, err
	}

	return s, nil
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
		BuildStage []string `yaml:"build_stage"`
	} `yaml:"config"`
}

func (s *Spack) readConfig() error {
	pr, pw := io.Pipe()
	cmd := s.exec("config", "get", "config")
	cmd.Stdout = pw

	go func() {
		cmd.Run()
		pw.Close()
	}()

	err := yaml.NewDecoder(pr).Decode(&s.config)
	if err != nil {
		return err
	}

	return nil
}

func (s *Spack) GetInstallRoot() string {
	return replaceVars(s.config.Config.InstallTree.Root)
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

func (i *Install) SpackPath() string {
	return i.Spec.Name + "-" + i.Spec.Version + "-" + i.Spec.Hash
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
	root := s.GetInstallRoot()

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

func (s *Spack) GetStageDir() string {
	for _, path := range s.config.Config.BuildStage {
		return replaceVars(path)
	}

	return ""
}

func (s *Spack) GetEnvVars(pkgs map[string]*Install) (*bytes.Buffer, error) {
	args := make([]string, 2, 2+len(pkgs))

	args[0] = "load"
	args[1] = "--sh"

	for _, pkg := range pkgs {
		args = append(args, pkg.Spec.Name+"/"+pkg.Spec.Hash)
	}

	buf := bytes.NewBuffer(nil)

	cmd := s.exec(args...)
	cmd.Stdout = buf

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return buf, nil
}
