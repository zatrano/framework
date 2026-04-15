package gen

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ModuleImportPath returns the Go module path from go.mod in moduleRoot.
func ModuleImportPath(moduleRoot string) (string, error) {
	p := filepath.Join(moduleRoot, "go.mod")
	b, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", p, err)
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "module ") {
			mod := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if mod == "" {
				return "", fmt.Errorf("empty module path in go.mod")
			}
			return mod, nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("module directive not found in go.mod")
}

