package generator

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// OverlayRequest applies a template tree onto an existing destination.
// It never deletes files. An existing file is replaced only when its bytes
// equal a known generator stub from BaseFS after substitution (bytes.Equal).
// Similar or user-edited files are skipped.
type OverlayRequest struct {
	FS            fs.FS
	Root          string
	Dest          string
	Substitutions map[string]string
	Skip          map[string]bool
	BaseFS        fs.FS
	BaseRoot      string
}

// OverlayResult reports per-file outcomes. Paths use slash separators.
type OverlayResult struct {
	Written   []string
	Identical []string
	Replaced  []string
	Skipped   []string
}

// Changed reports whether any file was created or stub-replaced.
func (r OverlayResult) Changed() bool {
	return len(r.Written)+len(r.Replaced) > 0
}

// Overlay copies Root onto an existing Dest. Dest must already exist.
func Overlay(req OverlayRequest) (OverlayResult, error) {
	var out OverlayResult
	if strings.TrimSpace(req.Dest) == "" {
		return out, fmt.Errorf("generator: destination required")
	}
	if strings.TrimSpace(req.Root) == "" {
		return out, fmt.Errorf("generator: template root required")
	}
	if req.FS == nil {
		return out, fmt.Errorf("generator: template filesystem required")
	}
	st, err := os.Stat(req.Dest)
	if err != nil {
		return out, fmt.Errorf("generator: destination must exist")
	}
	if !st.IsDir() {
		return out, fmt.Errorf("generator: destination must be a directory")
	}
	if err := rejectPathEscape(req.Root); err != nil {
		return out, err
	}
	root := strings.TrimSuffix(filepath.ToSlash(req.Root), "/")
	prefix := root + "/"
	baseRoot := strings.TrimSuffix(filepath.ToSlash(req.BaseRoot), "/")
	err = fs.WalkDir(req.FS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(filepath.ToSlash(path), prefix)
		if rel == "" || rel == root || path == root {
			return nil
		}
		if err := rejectPathEscape(rel); err != nil {
			return err
		}
		rel = StripTmplSuffix(rel)
		if req.Skip[rel] {
			return nil
		}
		destPath := filepath.Join(req.Dest, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}
		raw, err := fs.ReadFile(req.FS, path)
		if err != nil {
			return err
		}
		body := []byte(applySubs(string(raw), req.Substitutions))
		existing, err := os.ReadFile(destPath)
		if err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
					return err
				}
				mode := os.FileMode(0o644)
				if strings.HasSuffix(rel, ".sh") {
					mode = 0o755
				}
				if err := os.WriteFile(destPath, body, mode); err != nil {
					return err
				}
				out.Written = append(out.Written, rel)
				return nil
			}
			return err
		}
		if bytes.Equal(existing, body) {
			out.Identical = append(out.Identical, rel)
			return nil
		}
		if stub, ok := readKnownStub(req.BaseFS, baseRoot, rel, strings.HasSuffix(path, ".tmpl"), req.Substitutions); ok && bytes.Equal(existing, stub) {
			if err := os.WriteFile(destPath, body, 0o644); err != nil {
				return err
			}
			out.Replaced = append(out.Replaced, rel)
			return nil
		}
		out.Skipped = append(out.Skipped, rel)
		return nil
	})
	if err != nil {
		return out, err
	}
	sort.Strings(out.Written)
	sort.Strings(out.Identical)
	sort.Strings(out.Replaced)
	sort.Strings(out.Skipped)
	return out, nil
}

// readKnownStub returns the exact generated base-stub bytes for rel, or false
// if that path is not a known generator file. Matching is bytes.Equal only.
func readKnownStub(base fs.FS, baseRoot, rel string, overlayTmpl bool, subs map[string]string) ([]byte, bool) {
	if base == nil || baseRoot == "" || rel == "" {
		return nil, false
	}
	var candidates []string
	if overlayTmpl {
		candidates = append(candidates, baseRoot+"/"+rel+".tmpl")
	}
	candidates = append(candidates, baseRoot+"/"+rel)
	for _, p := range candidates {
		raw, err := fs.ReadFile(base, p)
		if err != nil {
			continue
		}
		return []byte(applySubs(string(raw), subs)), true
	}
	return nil, false
}
