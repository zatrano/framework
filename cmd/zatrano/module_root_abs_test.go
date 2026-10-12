package main

import (
	"path/filepath"
	"testing"
)

func moduleRootAbs(t *testing.T) (string, error) {
	t.Helper()
	return filepath.Abs(moduleRoot(t))
}
