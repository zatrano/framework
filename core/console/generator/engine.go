package generator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Known first-party scaffold names embedded in the CLI.
const (
	ScaffoldEmpty = "empty"
	ScaffoldAPI   = "api"
	ScaffoldWeb   = "web"
	ScaffoldFull  = "full"
	// ScaffoldApp is the only zatrano-new product: HTML / plus JSON /api.
	ScaffoldApp   = "app"
	LayoutVersion = "2"
)

// Request copies a template tree into Dest with placeholder substitution.
type Request struct {
	FS               fs.FS
	Root             string
	Dest             string
	Substitutions    map[string]string
	ScaffoldName     string
	ScaffoldVersion  string
	SkipScaffoldMeta bool
}

// Apply walks Root on FS and writes the consumer tree under Dest.
// It does not choose a scaffold, enable packages, or encode application policy.
func Apply(req Request) error {
	if strings.TrimSpace(req.Dest) == "" {
		return fmt.Errorf("generator: destination required")
	}
	if strings.TrimSpace(req.Root) == "" {
		return fmt.Errorf("generator: template root required")
	}
	if req.FS == nil {
		return fmt.Errorf("generator: template filesystem required")
	}
	if err := rejectPathEscape(req.Root); err != nil {
		return err
	}
	if _, err := os.Stat(req.Dest); err == nil {
		return fmt.Errorf("directory already exists: %s", req.Dest)
	}
	root := strings.TrimSuffix(filepath.ToSlash(req.Root), "/")
	prefix := root + "/"
	err := fs.WalkDir(req.FS, root, func(path string, d fs.DirEntry, err error) error {
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
		out := filepath.Join(req.Dest, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		raw, err := fs.ReadFile(req.FS, path)
		if err != nil {
			return err
		}
		body := applySubs(string(raw), req.Substitutions)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(rel, ".sh") {
			mode = 0o755
		}
		return os.WriteFile(out, []byte(body), mode)
	})
	if err != nil {
		return err
	}
	if req.SkipScaffoldMeta {
		return nil
	}
	digest, err := Digest(req.FS, root)
	if err != nil {
		return err
	}
	return writeScaffoldMeta(req, digest)
}

// StripTmplSuffix removes a trailing .tmpl so go.mod.tmpl becomes go.mod.
func StripTmplSuffix(rel string) string {
	return strings.TrimSuffix(rel, ".tmpl")
}

func applySubs(body string, subs map[string]string) string {
	for old, neu := range subs {
		if old == "" {
			continue
		}
		body = strings.ReplaceAll(body, old, neu)
	}
	return body
}

func rejectPathEscape(rel string) error {
	slash := filepath.ToSlash(rel)
	if strings.Contains(slash, "..") {
		return fmt.Errorf("invalid template path")
	}
	return nil
}
