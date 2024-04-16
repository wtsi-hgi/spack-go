package spack

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"

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
	return exec.Command(s.exe, append(append([]string{}, s.args...), args...)...)
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
	cmd.Stderr = os.Stderr

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
	cmd.Stderr = os.Stderr

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
