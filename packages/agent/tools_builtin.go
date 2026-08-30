package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zatrano/framework/packages/ai"
)

// WebFetchOptions configures the built-in web_fetch tool.
type WebFetchOptions struct {
	Name       string        // tool name; default web_fetch
	MaxBytes   int           // default 100_000
	Timeout    time.Duration // default 15s
	HTTPClient *http.Client
	AllowHTTP  bool // default false (HTTPS only)
}

// RegisterWebFetch registers a tool that GETs a URL and returns truncated text.
func RegisterWebFetch(reg *Registry, opts WebFetchOptions) error {
	if reg == nil {
		return fmt.Errorf("agent: registry is nil")
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = "web_fetch"
	}
	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 100_000
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	params := json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","description":"HTTPS URL to fetch"}},"required":["url"]}`)
	tool := ai.FunctionTool(name, "Fetch a web page and return its text body (truncated).", params)
	return reg.Register(tool, func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args struct {
			URL string `json:"url"`
		}
		if err := call.UnmarshalArguments(&args); err != nil {
			return "", err
		}
		u := strings.TrimSpace(args.URL)
		if u == "" {
			return "", fmt.Errorf("url is required")
		}
		if !opts.AllowHTTP && !strings.HasPrefix(strings.ToLower(u), "https://") {
			return "", fmt.Errorf("only https URLs are allowed")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", "ZATRANO-agent/1.6")
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		limited := io.LimitReader(resp.Body, int64(maxBytes)+1)
		raw, err := io.ReadAll(limited)
		if err != nil {
			return "", err
		}
		truncated := len(raw) > maxBytes
		if truncated {
			raw = raw[:maxBytes]
		}
		text := string(raw)
		if !utf8.ValidString(text) {
			text = strings.ToValidUTF8(text, "")
		}
		out, _ := json.Marshal(map[string]any{
			"status":    resp.StatusCode,
			"truncated": truncated,
			"bytes":     len(raw),
			"body":      text,
		})
		return string(out), nil
	})
}

// FileSearchOptions configures the built-in file_search tool.
type FileSearchOptions struct {
	Name       string   // default file_search
	Root       string   // required; searches stay under this directory
	MaxFiles   int      // default 20
	MaxBytes   int      // max bytes read per matched file; default 16_000
	Extensions []string // optional allow-list (e.g. .md, .go); empty = all text-ish
}

// RegisterFileSearch registers a tool that finds files under Root matching a query substring / glob.
func RegisterFileSearch(reg *Registry, opts FileSearchOptions) error {
	if reg == nil {
		return fmt.Errorf("agent: registry is nil")
	}
	root := filepath.Clean(opts.Root)
	if root == "" || root == "." {
		return fmt.Errorf("agent: FileSearch Root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = "file_search"
	}
	maxFiles := opts.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 20
	}
	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 16_000
	}
	extOK := map[string]bool{}
	for _, e := range opts.Extensions {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		extOK[e] = true
	}
	params := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Substring or glob (e.g. *.md) to match paths/content"},"glob":{"type":"string","description":"Optional path glob relative to root"}},"required":["query"]}`)
	tool := ai.FunctionTool(name, "Search files under a sandboxed root and return matching snippets.", params)
	return reg.Register(tool, func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args struct {
			Query string `json:"query"`
			Glob  string `json:"glob"`
		}
		if err := call.UnmarshalArguments(&args); err != nil {
			return "", err
		}
		q := strings.TrimSpace(args.Query)
		if q == "" {
			return "", fmt.Errorf("query is required")
		}
		pattern := strings.TrimSpace(args.Glob)
		type hit struct {
			Path    string `json:"path"`
			Snippet string `json:"snippet,omitempty"`
		}
		var hits []hit
		err := filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if len(hits) >= maxFiles {
				return filepath.SkipAll
			}
			rel, err := filepath.Rel(absRoot, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if len(extOK) > 0 && !extOK[strings.ToLower(filepath.Ext(path))] {
				return nil
			}
			if pattern != "" {
				ok, _ := filepath.Match(pattern, filepath.Base(path))
				ok2, _ := filepath.Match(pattern, rel)
				if !ok && !ok2 {
					return nil
				}
			}
			nameMatch := strings.Contains(strings.ToLower(rel), strings.ToLower(q))
			snippet := ""
			contentMatch := false
			raw, rerr := os.ReadFile(path)
			if rerr == nil {
				if len(raw) > maxBytes {
					raw = raw[:maxBytes]
				}
				text := string(raw)
				if !utf8.ValidString(text) {
					text = strings.ToValidUTF8(text, "")
				}
				if strings.Contains(strings.ToLower(text), strings.ToLower(q)) {
					contentMatch = true
					snippet = truncateSnippet(text, q, 240)
				}
			}
			if nameMatch || contentMatch || (pattern != "" && q == "*") {
				hits = append(hits, hit{Path: rel, Snippet: snippet})
			}
			return nil
		})
		if err != nil && err != filepath.SkipAll {
			return "", err
		}
		out, _ := json.Marshal(map[string]any{"root": absRoot, "hits": hits, "count": len(hits)})
		return string(out), nil
	})
}

func truncateSnippet(text, query string, max int) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, strings.ToLower(query))
	if idx < 0 {
		if len(text) > max {
			return text[:max]
		}
		return text
	}
	start := idx - 40
	if start < 0 {
		start = 0
	}
	end := start + max
	if end > len(text) {
		end = len(text)
	}
	s := text[start:end]
	if start > 0 {
		s = "…" + s
	}
	if end < len(text) {
		s += "…"
	}
	return s
}
