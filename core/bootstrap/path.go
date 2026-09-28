package bootstrap

import (
	"os"
	"path/filepath"
)

func lookWorkingDirectory() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return ".", err
	}

	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd, nil
		}
		dir = parent
	}
}
