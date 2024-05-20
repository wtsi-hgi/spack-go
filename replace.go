package spack

import (
	"os"
	"os/user"
	"path/filepath"
	"time"
)

var replaceable = map[string]func() string{
	"$tempdir":          getTmpDir,
	"$user":             getUser,
	"$architecture":     getArch,
	"$arch":             getArch,
	"$platform":         getPlatform,
	"$os":               getOS,
	"$operating_system": getOS,
	"$target":           getTarget,
	"$target_family":    getTargetFamily,
	"$date":             getDate,
}

func replaceVars(path string) string {
	if path == "" || path == "/" || path == "." {
		return ""
	}

	file := filepath.Base(path)
	dir := filepath.Dir(path)

	if fn, ok := replaceable[file]; ok {
		file = fn()
	}

	return filepath.Join(replaceVars(dir), file)
}

var tmpVars = [...]string{"TMPDIR", "TEMP", "TMP"}

func getTmpDir() string {
	for _, env := range tmpVars {
		path := os.Getenv(env)

		if path != "" && isUserWritable(path) {
			return path
		}
	}

	for _, path := range tmpDirs {
		if isUserWritable(path) {
			return path
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return "."
	}

	return wd
}

func isUserWritable(path string) bool {
	f, err := os.CreateTemp(path, "")
	if err != nil {
		return false
	}

	f.Close()
	os.Remove(f.Name())

	return true
}

func getUser() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}

	return u.Username
}

func getArch() string {
	return ""
}

func getPlatform() string {
	return ""
}

func getOS() string {
	return ""
}

func getTarget() string {
	return ""
}

func getTargetFamily() string {
	return ""
}

func getDate() string {
	return time.Now().Format(time.DateOnly)
}
