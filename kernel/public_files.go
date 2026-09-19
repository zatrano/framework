package kernel

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/zatrano/framework/v2/kernel/safepath"
)

// publicFileIndex is a production-only negative filter for Application.publicFile.
// Hits still take the Resolve + EvalUnder + Stat path (defense in depth).
//
// Files created under non-symlink directories after boot are not served until
// restart. Symlink entries are recorded as dynamic prefixes so upload trees
// (public/storage → another directory) still use the slow path.
type publicFileIndex struct {
	files           map[string]struct{}
	dynamicPrefixes []string
}

func (app *Application) ensurePublicFileIndex() {
	if app == nil || !app.IsProduction() {
		return
	}
	app.publicFilesOnce.Do(func() {
		app.publicFiles = buildPublicFileIndex(app.BasePath("public"))
	})
}

func buildPublicFileIndex(root string) *publicFileIndex {
	idx := &publicFileIndex{files: make(map[string]struct{})}
	if root == "" {
		return idx
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return idx
	}
	_ = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path != abs {
				if key, ok := publicURLFromDisk(abs, path); ok {
					idx.addDynamic(key)
				}
			}
			return nil
		}
		if path == abs {
			return nil
		}
		key, ok := publicURLFromDisk(abs, path)
		if !ok {
			return nil
		}
		// Windows junctions often show ModeIrregular, not ModeSymlink.
		if entryIsSymlink(d, path) || d.Type()&fs.ModeIrregular != 0 {
			info, statErr := os.Stat(path)
			if statErr != nil {
				idx.addDynamic(key)
				return filepath.SkipDir
			}
			if info.IsDir() {
				idx.addDynamic(key)
				return filepath.SkipDir
			}
			idx.addFile(key)
			return nil
		}
		if d.IsDir() {
			if dirLeavesPublicTree(abs, path) {
				idx.addDynamic(key)
				return filepath.SkipDir
			}
			return nil
		}
		idx.addFile(key)
		return nil
	})
	return idx
}

func publicURLFromDisk(root, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	rel = filepath.Clean(rel)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return foldPublicKey("/" + filepath.ToSlash(rel)), true
}

func (idx *publicFileIndex) addFile(key string) {
	if idx == nil || key == "" || key == "/" {
		return
	}
	idx.files[key] = struct{}{}
}

func (idx *publicFileIndex) addDynamic(key string) {
	if idx == nil || key == "" || key == "/" {
		return
	}
	for _, existing := range idx.dynamicPrefixes {
		if existing == key {
			return
		}
	}
	idx.dynamicPrefixes = append(idx.dynamicPrefixes, key)
}

func (idx *publicFileIndex) mayServe(key string) bool {
	if idx == nil || key == "" {
		return false
	}
	if _, ok := idx.files[key]; ok {
		return true
	}
	for _, prefix := range idx.dynamicPrefixes {
		if key == prefix || strings.HasPrefix(key, prefix+"/") {
			return true
		}
	}
	return false
}

// publicFileLookupKey mirrors safepath.Resolve cleaning without joining the
// public root, so index lookup matches req.Path() after the same rejection
// rules (null byte, "..", absolute/UNC, Windows separators).
func publicFileLookupKey(userPath string) (string, bool) {
	if strings.ContainsRune(userPath, 0) {
		return "", false
	}
	slash := strings.ReplaceAll(userPath, `\`, "/")
	slash = strings.TrimPrefix(slash, "/")
	if slash == "" || slash == "." {
		return "", false
	}
	if strings.Contains(slash, ":") || strings.HasPrefix(slash, "//") {
		return "", false
	}
	for _, part := range strings.Split(slash, "/") {
		if part == ".." {
			return "", false
		}
	}
	rel := filepath.FromSlash(slash)
	rel = filepath.Clean(rel)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return foldPublicKey("/" + filepath.ToSlash(rel)), true
}

func foldPublicKey(key string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(key)
	}
	return key
}

func entryIsSymlink(d fs.DirEntry, path string) bool {
	if d != nil && d.Type()&fs.ModeSymlink != 0 {
		return true
	}
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

// dirLeavesPublicTree detects Windows junctions / reparse points that WalkDir
// may list as ordinary directories without ModeSymlink.
// Compare resolved paths against the public root, not Abs(path)==Eval(path):
// CI runners often canonicalize Temp/checkout via a junction, so those strings
// differ even for a normal nested directory.
func dirLeavesPublicTree(root, path string) bool {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		resolvedRoot = root
	}
	eval, err := filepath.EvalSymlinks(path)
	if err != nil {
		info, stErr := os.Lstat(path)
		if stErr != nil || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			return true
		}
		return false
	}
	return !safepath.Under(resolvedRoot, eval)
}
