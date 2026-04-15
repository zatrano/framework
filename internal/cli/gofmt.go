package cli

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// goFmtWireFile runs `go fmt` on a single file relative to moduleRoot (requires Go toolchain on PATH).
func goFmtWireFile(moduleRoot, wireFile string) error {
	moduleRoot = filepath.Clean(moduleRoot)
	rel, err := filepath.Rel(moduleRoot, filepath.Clean(wireFile))
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	cmd := exec.Command("go", "fmt", rel)
	cmd.Dir = moduleRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}

