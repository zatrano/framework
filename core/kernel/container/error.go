package container

import (
	"fmt"
	"strings"
)

// ResolutionKind classifies a container Make failure.
type ResolutionKind string

const (
	KindUnbound        ResolutionKind = "unbound"
	KindInvalidFactory ResolutionKind = "invalid_factory"
	KindCircular       ResolutionKind = "circular"
	KindAliasCycle     ResolutionKind = "alias_cycle"
	KindResolveFailed  ResolutionKind = "resolve_failed"
	KindNilContainer   ResolutionKind = "nil_container"
)

// ResolutionError is a kernel-private Make failure with provenance.
// Cause is the original error (never a formatted dump of it). Unwrap
// preserves errors.Is / errors.As. Exact Error() text is not API.
type ResolutionError struct {
	Kind   ResolutionKind
	Target string
	Path   []string
	Cause  error
}

func (e *ResolutionError) Error() string {
	if e == nil {
		return "container: resolution error"
	}
	path := strings.Join(e.Path, " -> ")
	switch {
	case e.Cause != nil && path != "":
		return fmt.Sprintf("container: %s: %s: %v", e.Kind, path, e.Cause)
	case e.Cause != nil:
		return fmt.Sprintf("container: %s: %v", e.Kind, e.Cause)
	case path != "":
		return fmt.Sprintf("container: %s: %s", e.Kind, path)
	case e.Target != "":
		return fmt.Sprintf("container: %s: %q", e.Kind, e.Target)
	default:
		return fmt.Sprintf("container: %s", e.Kind)
	}
}

func (e *ResolutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func annotate(err error, target string, path []string) error {
	if err == nil {
		return nil
	}
	path = clonePath(path)
	if re, ok := err.(*ResolutionError); ok {
		cp := *re
		cp.Target = target
		if len(cp.Path) == 0 {
			cp.Path = path
		} else {
			cp.Path = mergePath(path, cp.Path)
		}
		return &cp
	}
	return &ResolutionError{
		Kind:   KindResolveFailed,
		Target: target,
		Path:   path,
		Cause:  err,
	}
}

func clonePath(path []string) []string {
	if len(path) == 0 {
		return nil
	}
	out := make([]string, len(path))
	copy(out, path)
	return out
}

func appendPath(path []string, hops ...string) []string {
	return mergePath(path, hops)
}

func mergePath(prefix, suffix []string) []string {
	if len(prefix) == 0 {
		return clonePath(suffix)
	}
	if len(suffix) == 0 {
		return clonePath(prefix)
	}
	if len(suffix) >= len(prefix) && equalPath(suffix[:len(prefix)], prefix) {
		return clonePath(suffix)
	}
	max := len(prefix)
	if len(suffix) < max {
		max = len(suffix)
	}
	for n := max; n > 0; n-- {
		if equalPath(prefix[len(prefix)-n:], suffix[:n]) {
			out := make([]string, len(prefix)+len(suffix)-n)
			copy(out, prefix)
			copy(out[len(prefix):], suffix[n:])
			return out
		}
	}
	out := make([]string, len(prefix)+len(suffix))
	copy(out, prefix)
	copy(out[len(prefix):], suffix)
	return out
}

func equalPath(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
