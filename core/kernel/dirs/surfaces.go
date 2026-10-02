package dirs

import (
	"path/filepath"
	"strings"
	"unicode"
)

const (
	SurfaceWeb  = "web"
	SurfaceAPI  = "api"
	SurfaceAuth = "auth"
)

// ReservedHTTPSurfaces cannot be panel names. They are the built-in HTTP surfaces.
func ReservedHTTPSurfaces() []string {
	return []string{SurfaceWeb, SurfaceAPI, SurfaceAuth}
}

// ValidPanelName is a generated HTML surface (make:panel {name}).
func ValidPanelName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || len(name) > 64 {
		return false
	}
	for _, reserved := range ReservedHTTPSurfaces() {
		if name == reserved {
			return false
		}
	}
	for i, r := range name {
		if i == 0 {
			if r < 'a' || r > 'z' {
				return false
			}
			continue
		}
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func firstPathSegment(rel, prefix string) (string, bool) {
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(rel, prefix)
	if rest == "" {
		return "", false
	}
	seg, _, _ := strings.Cut(rest, "/")
	if seg == "" || strings.Contains(seg, "..") {
		return "", false
	}
	for _, r := range seg {
		if r > unicode.MaxASCII || (!unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_') {
			return "", false
		}
	}
	return strings.ToLower(seg), true
}

// RouteSurface is the first directory under app/routes/.
func RouteSurface(rel string) (string, bool) {
	return firstPathSegment(rel, "app/routes/")
}

// ControllerSurface is the first directory under app/http/handlers/.
func ControllerSurface(rel string) (string, bool) {
	return firstPathSegment(rel, "app/http/handlers/")
}

func surfaceAllowed(name string) bool {
	switch name {
	case SurfaceWeb, SurfaceAPI, SurfaceAuth:
		return true
	default:
		return ValidPanelName(name)
	}
}

// RouteRelAllowed is where RegisterWeb / RegisterAPI / HTTP verbs may live.
func RouteRelAllowed(rel string) bool {
	name, ok := RouteSurface(rel)
	return ok && surfaceAllowed(name)
}

// HandlerRelAllowed is where *Controller types may live.
func HandlerRelAllowed(rel string) bool {
	name, ok := ControllerSurface(rel)
	return ok && surfaceAllowed(name)
}

// HandlerKind is the HTTP transport of a controller file.
// Auth JSON lives under handlers/auth/api (surface is still auth).
func HandlerKind(rel string) string {
	rel = filepath.ToSlash(rel)
	switch {
	case strings.HasPrefix(rel, "app/http/handlers/api/"),
		strings.HasPrefix(rel, "app/http/handlers/auth/api/"):
		return SurfaceAPI
	case strings.HasPrefix(rel, "app/http/handlers/web/"),
		strings.HasPrefix(rel, "app/http/handlers/auth/web/"):
		return SurfaceWeb
	default:
		name, ok := ControllerSurface(rel)
		if !ok {
			return ""
		}
		return name
	}
}
