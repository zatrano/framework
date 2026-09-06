package safepath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxSymlinkHops = 32

// Under reports whether candidate resolves inside root (inclusive).
func Under(root, candidate string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absCand, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	absRoot = stripLongPath(absRoot)
	absCand = stripLongPath(absCand)
	rel, err := filepath.Rel(absRoot, absCand)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func stripLongPath(p string) string {
	const prefix = `\\?\`
	if strings.HasPrefix(p, prefix) {
		p = p[len(prefix):]
		if len(p) >= 4 && (p[:4] == `UNC\` || p[:4] == `unc\`) {
			p = `\\` + p[4:]
		}
	}
	return p
}

// Resolve joins root with a user-supplied relative path and rejects escapes.
// Absolute user paths, ".." segments that leave root, and null bytes are rejected.
// Both "/" and "\" are treated as separators on every OS (upload/path spoofing).
func Resolve(root, userPath string) (string, error) {
	if strings.ContainsRune(userPath, 0) {
		return "", fmt.Errorf("path contains null byte")
	}
	slash := strings.ReplaceAll(userPath, `\`, "/")
	slash = strings.TrimPrefix(slash, "/")
	if slash == "" || slash == "." {
		full, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		return full, nil
	}
	// Reject Windows drive / UNC style after separator normalization.
	if strings.Contains(slash, ":") || strings.HasPrefix(slash, "//") {
		return "", fmt.Errorf("absolute path rejected")
	}
	for _, part := range strings.Split(slash, "/") {
		if part == ".." {
			return "", fmt.Errorf("path escapes root")
		}
	}
	rel := filepath.FromSlash(slash)
	rel = filepath.Clean(rel)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes root")
	}
	full := filepath.Join(root, rel)
	if !Under(root, full) {
		return "", fmt.Errorf("path escapes root")
	}
	return full, nil
}

// EvalUnder symlink-resolves candidate and root, then requires the resolved
// target to stay inside the resolved root. EvalSymlinks alone is not the
// invariant; Under(resolvedRoot, resolved) is.
//
// Windows directory junctions are followed via Readlink because
// filepath.EvalSymlinks does not walk them. Serve the returned path (the
// canonical target), not the original candidate, so swapping a request-path
// symlink after this check does not change what is opened. Residual TOCTOU
// remains if the canonical target itself is replaced between this check and
// Open (kernel has no portable O_NOFOLLOW).
func EvalUnder(root, candidate string) (string, error) {
	resolvedRoot, err := resolveFS(root)
	if err != nil {
		return "", err
	}
	resolved, err := resolveFS(candidate)
	if err != nil {
		return "", err
	}
	if !Under(resolvedRoot, resolved) {
		return "", fmt.Errorf("path escapes root")
	}
	return resolved, nil
}

func resolveFS(path string) (string, error) {
	return resolveFSN(path, 0)
}

func resolveFSN(path string, depth int) (string, error) {
	if depth > maxSymlinkHops {
		return "", fmt.Errorf("too many symlinks")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if eval, err := filepath.EvalSymlinks(abs); err == nil {
		return stripLongPath(eval), nil
	}
	base, parts := splitAbs(abs)
	current := base
	for _, part := range parts {
		next := filepath.Join(current, part)
		target, err := os.Readlink(next)
		if err == nil {
			if !filepath.IsAbs(target) {
				target = filepath.Join(current, target)
			}
			resolved, err := resolveFSN(target, depth+1)
			if err != nil {
				return "", err
			}
			current = resolved
			continue
		}
		if _, err := os.Lstat(next); err != nil {
			return "", err
		}
		current = next
	}
	return current, nil
}

func splitAbs(abs string) (base string, parts []string) {
	vol := filepath.VolumeName(abs)
	rest := abs[len(vol):]
	sep := string(filepath.Separator)
	rest = strings.TrimPrefix(rest, sep)
	rest = strings.ReplaceAll(rest, `\`, "/")
	for _, p := range strings.Split(rest, "/") {
		if p == "" || p == "." {
			continue
		}
		parts = append(parts, p)
	}
	if vol != "" {
		return vol + sep, parts
	}
	return sep, parts
}
