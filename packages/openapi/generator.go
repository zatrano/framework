package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/packages/routing"
)

// Spec is a minimal OpenAPI 3 document.
type Spec struct {
	OpenAPI string         `json:"openapi"`
	Info    Info           `json:"info"`
	Servers []Server       `json:"servers,omitempty"`
	Paths   map[string]any `json:"paths"`
}

// Info holds API metadata.
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// Server describes an API server.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// Options configures generation.
type Options struct {
	Title       string
	Description string
	Version     string
	ServerURL   string
}

// Generate builds an OpenAPI document from registered routes.
func Generate(routes []*routing.Route, opts Options) *Spec {
	if opts.Title == "" {
		opts.Title = "ZATRANO API"
	}
	if opts.Version == "" {
		opts.Version = "1.0.0"
	}
	spec := &Spec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       opts.Title,
			Description: opts.Description,
			Version:     opts.Version,
		},
		Paths: map[string]any{},
	}
	if opts.ServerURL != "" {
		spec.Servers = []Server{{URL: opts.ServerURL, Description: "Application"}}
	}

	for _, route := range routes {
		if route == nil {
			continue
		}
		path := toOpenAPIPath(route.Path)
		item, _ := spec.Paths[path].(map[string]any)
		if item == nil {
			item = map[string]any{}
		}
		operation := map[string]any{
			"summary": route.Name,
			"responses": map[string]any{
				"200": map[string]any{
					"description": "Successful response",
				},
			},
		}
		if route.Name != "" {
			operation["operationId"] = strings.ReplaceAll(route.Name, ".", "_")
			operation["tags"] = []string{tagFor(route.Name, route.Path)}
		}
		params := pathParams(route.Path)
		if len(params) > 0 {
			operation["parameters"] = params
		}
		method := strings.ToLower(route.Method)
		if method == "head" || method == "options" {
			continue
		}
		item[method] = operation
		spec.Paths[path] = item
	}
	return spec
}

// WriteJSON writes the OpenAPI document to path.
func WriteJSON(path string, spec *Spec) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// WriteYAML writes a simple YAML representation (no external deps).
func WriteYAML(path string, spec *Spec) error {
	raw, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	// Prefer JSON content with .yaml only when requested; keep valid JSON subset.
	// Callers wanting real YAML can convert; we emit readable JSON-compatible YAML-ish.
	var indented any
	if err := json.Unmarshal(raw, &indented); err != nil {
		return err
	}
	out := toYAML(indented, 0)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(out), 0o644)
}

func toOpenAPIPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			continue
		}
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + strings.TrimPrefix(part, ":") + "}"
		}
	}
	return strings.Join(parts, "/")
}

func pathParams(path string) []map[string]any {
	out := make([]map[string]any, 0)
	for _, part := range strings.Split(path, "/") {
		name := ""
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			name = strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
		} else if strings.HasPrefix(part, ":") {
			name = strings.TrimPrefix(part, ":")
		}
		if name == "" {
			continue
		}
		out = append(out, map[string]any{
			"name":     name,
			"in":       "path",
			"required": true,
			"schema":   map[string]any{"type": "string"},
		})
	}
	return out
}

func tagFor(name, path string) string {
	if name != "" {
		parts := strings.Split(name, ".")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "default"
}

func toYAML(value any, indent int) string {
	pad := strings.Repeat("  ", indent)
	switch v := value.(type) {
	case map[string]any:
		if len(v) == 0 {
			return "{}"
		}
		var b strings.Builder
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		// stable-ish: openapi, info, servers, paths first when present
		priority := []string{"openapi", "info", "servers", "paths", "title", "description", "version", "url"}
		ordered := make([]string, 0, len(keys))
		seen := map[string]bool{}
		for _, key := range priority {
			if _, ok := v[key]; ok {
				ordered = append(ordered, key)
				seen[key] = true
			}
		}
		for _, key := range keys {
			if !seen[key] {
				ordered = append(ordered, key)
			}
		}
		for i, key := range ordered {
			child := v[key]
			if i > 0 {
				b.WriteString("\n")
			}
			switch child.(type) {
			case map[string]any, []any:
				b.WriteString(pad)
				b.WriteString(key)
				b.WriteString(":\n")
				nested := toYAML(child, indent+1)
				b.WriteString(nested)
			default:
				b.WriteString(pad)
				b.WriteString(key)
				b.WriteString(": ")
				b.WriteString(yamlScalar(child))
			}
		}
		return b.String()
	case []any:
		if len(v) == 0 {
			return pad + "[]"
		}
		var b strings.Builder
		for i, item := range v {
			if i > 0 {
				b.WriteString("\n")
			}
			switch item.(type) {
			case map[string]any, []any:
				b.WriteString(pad)
				b.WriteString("-\n")
				b.WriteString(toYAML(item, indent+1))
			default:
				b.WriteString(pad)
				b.WriteString("- ")
				b.WriteString(yamlScalar(item))
			}
		}
		return b.String()
	default:
		return pad + yamlScalar(v)
	}
}

func yamlScalar(value any) string {
	switch v := value.(type) {
	case string:
		if strings.ContainsAny(v, ":#\n\"'") || v == "" {
			raw, _ := json.Marshal(v)
			return string(raw)
		}
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%v", v)
	case nil:
		return "null"
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(raw)
	}
}
