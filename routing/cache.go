package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// RouteInfo is a serializable route snapshot (handlers are not cached).
type RouteInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Name   string `json:"name,omitempty"`
}

// Snapshot returns route metadata for listing/caching.
func (r *Router) Snapshot() []RouteInfo {
	routes := r.Routes()
	out := make([]RouteInfo, 0, len(routes))
	for _, route := range routes {
		out = append(out, RouteInfo{
			Method: route.Method,
			Path:   route.Path,
			Name:   route.Name,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].Method < out[j].Method
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// SaveCache writes route metadata to path.
func (r *Router) SaveCache(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(r.Snapshot(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// LoadRouteCache reads cached route metadata.
func LoadRouteCache(path string) ([]RouteInfo, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []RouteInfo
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// ClearRouteCache removes the route cache file.
func ClearRouteCache(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
