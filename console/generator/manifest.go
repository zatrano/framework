package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Manifest is the first-party scaffold contract recorded into a generated app.
// It is metadata only: runtime bootstrap does not read it.
type Manifest struct {
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	FrameworkMin     string   `json:"framework_min"`
	FrameworkMax     string   `json:"framework_max,omitempty"`
	RequiredPackages []string `json:"required_packages,omitempty"`
	LayoutVersion    string   `json:"layout_version"`
	Digest           string   `json:"digest"`
}

// Digest is a stable sha256 of the template tree (paths + raw bytes).
func Digest(fsys fs.FS, root string) (string, error) {
	var files []string
	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, filepath.ToSlash(path))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	h := sha256.New()
	for _, path := range files {
		raw, err := fs.ReadFile(fsys, path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\n%d\n", path, len(raw))
		_, _ = h.Write(raw)
		_, _ = h.Write([]byte{'\n'})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// CombinedDigest is a stable hash of already-computed digest strings.
func CombinedDigest(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		fmt.Fprintln(h, p)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// WriteScaffoldMeta records generator metadata. Runtime bootstrap does not read it.
func WriteScaffoldMeta(dest, name, version, digest string) error {
	return writeScaffoldMeta(Request{Dest: dest, ScaffoldName: name, ScaffoldVersion: version}, digest)
}

func writeScaffoldMeta(req Request, digest string) error {
	name := strings.TrimSpace(req.ScaffoldName)
	if name == "" {
		name = filepath.Base(strings.TrimSuffix(filepath.ToSlash(req.Root), "/"))
	}
	ver := strings.TrimSpace(req.ScaffoldVersion)
	if ver == "" {
		ver = "unknown"
	}
	body := fmt.Sprintf(`package bootstrap

// Scaffold metadata recorded by zatrano new. Not consulted at runtime.
// Framework upgrades do not regenerate this file.
const (
	ScaffoldName      = %q
	ScaffoldVersion   = %q
	ScaffoldDigest    = %q
	ScaffoldLayoutVer = %q
)
`, name, ver, digest, LayoutVersion)
	return WriteFile(filepath.Join(req.Dest, "bootstrap", "scaffold.go"), body)
}
