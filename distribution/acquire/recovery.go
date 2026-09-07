package acquire

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	goModName = "go.mod"
	goSumName = "go.sum"
)

// RecoveryKind is SPEC §16: unavailable vs best-effort file restore vs
// restore failure. There is no guaranteed / transactional kind.
type RecoveryKind string

const (
	RecoveryUnavailable RecoveryKind = "unavailable"
	RecoveryFiles       RecoveryKind = "files"
	RecoveryFailed      RecoveryKind = "failed"
)

// Recovery is what file restore actually did. It is not a claim that Go
// module acquisition was fully rolled back.
type Recovery struct {
	Kind          RecoveryKind
	RestoredGoMod bool
	RestoredGoSum bool
	Err           error
}

// FileSnapshot is the raw go.mod / go.sum bytes in one module root.
// It is not parsed module graph state and not the module cache.
type FileSnapshot struct {
	Root         string
	GoMod        []byte
	GoSum        []byte
	GoModMissing bool
	GoSumMissing bool
}

// SnapshotFiles reads go.mod and go.sum. It does not take the mutation
// lock or write those files.
func SnapshotFiles(root string) (FileSnapshot, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return FileSnapshot{}, fmt.Errorf("acquire: module root required")
	}
	snap := FileSnapshot{Root: root}
	mod, err := os.ReadFile(filepath.Join(root, goModName))
	switch {
	case err == nil:
		snap.GoMod = cloneBytes(mod)
	case os.IsNotExist(err):
		snap.GoModMissing = true
	default:
		return FileSnapshot{}, err
	}
	sum, err := os.ReadFile(filepath.Join(root, goSumName))
	switch {
	case err == nil:
		snap.GoSum = cloneBytes(sum)
	case os.IsNotExist(err):
		snap.GoSumMissing = true
	default:
		return FileSnapshot{}, err
	}
	return snap, nil
}

// RecoverFiles writes the snapshot's go.mod and go.sum back under the
// per-root mutation lock. It does not undo the module cache, sumdb, or
// other Go tooling side effects, or change ApplyResult reports.
func RecoverFiles(ctx context.Context, snap FileSnapshot) (Recovery, error) {
	root := strings.TrimSpace(snap.Root)
	rec := Recovery{Kind: RecoveryFailed}
	if root == "" {
		rec.Err = fmt.Errorf("acquire: module root required")
		return rec, rec.Err
	}
	unlock, err := lockMutation(ctx, root)
	if err != nil {
		rec.Err = err
		return rec, err
	}
	defer unlock()
	if err := restoreNamed(root, goModName, snap.GoMod, snap.GoModMissing); err != nil {
		rec.Err = err
		return rec, err
	}
	rec.RestoredGoMod = true
	if err := restoreNamed(root, goSumName, snap.GoSum, snap.GoSumMissing); err != nil {
		rec.Err = err
		return rec, err
	}
	rec.RestoredGoSum = true
	rec.Kind = RecoveryFiles
	rec.Err = nil
	return rec, nil
}

func restoreNamed(root, name string, data []byte, missing bool) error {
	if name != goModName && name != goSumName {
		return fmt.Errorf("acquire: recovery writes only go.mod and go.sum")
	}
	path := filepath.Join(root, name)
	if missing {
		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return os.WriteFile(path, cloneBytes(data), 0o644)
}

func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
