package console

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func packageEnvSectionMarkers(name string) (start, end string) {
	name = strings.ToLower(strings.TrimSpace(name))
	return "# --- zatrano:package:" + name + " ---", "# --- zatrano:package:" + name + ":end ---"
}

// mergePackageEnvFile appends missing keys from snippet into path.
// Existing keys (including commented KEY= lines) are left unchanged.
// A package section is written at most once.
func mergePackageEnvFile(path, name, snippet string) (added int, err error) {
	snippet = strings.ReplaceAll(snippet, "\r\n", "\n")
	snippet = strings.ReplaceAll(snippet, "\r", "\n")
	snippet = strings.TrimSpace(snippet)
	if snippet == "" || strings.TrimSpace(name) == "" {
		return 0, nil
	}
	start, end := packageEnvSectionMarkers(name)
	var existing []byte
	if b, readErr := os.ReadFile(path); readErr == nil {
		existing = b
	} else if !os.IsNotExist(readErr) {
		return 0, readErr
	}
	text := strings.ReplaceAll(string(existing), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if strings.Contains(text, start) {
		return 0, nil
	}
	have := envAssignmentKeys(text)
	block, n := filterEnvSnippet(snippet, have)
	if n == 0 {
		return 0, nil
	}
	var b strings.Builder
	trimmed := strings.TrimRight(text, "\n")
	if trimmed != "" {
		b.WriteString(trimmed)
		b.WriteString("\n\n")
	}
	b.WriteString(start)
	b.WriteByte('\n')
	b.WriteString(block)
	if !strings.HasSuffix(block, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString(end)
	b.WriteByte('\n')
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return 0, err
		}
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return 0, err
	}
	return n, nil
}

func filterEnvSnippet(snippet string, have map[string]bool) (string, int) {
	if have == nil {
		have = map[string]bool{}
	}
	var out strings.Builder
	var pending []string
	added := 0
	for _, line := range strings.Split(snippet, "\n") {
		key := envAssignmentKey(line)
		if key == "" {
			pending = append(pending, line)
			continue
		}
		if have[key] {
			pending = nil
			continue
		}
		for _, p := range pending {
			out.WriteString(p)
			out.WriteByte('\n')
		}
		pending = nil
		out.WriteString(line)
		out.WriteByte('\n')
		have[key] = true
		added++
	}
	return out.String(), added
}

func envAssignmentKeys(text string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		if key := envAssignmentKey(line); key != "" {
			out[key] = true
		}
	}
	return out
}

func envAssignmentKey(line string) string {
	s := strings.TrimSpace(line)
	if s == "" || strings.HasPrefix(s, "# --- zatrano:package:") {
		return ""
	}
	if strings.HasPrefix(s, "#") {
		s = strings.TrimSpace(s[1:])
	}
	if strings.HasPrefix(s, "export") && (len(s) == 6 || unicode.IsSpace(rune(s[6]))) {
		s = strings.TrimSpace(s[6:])
	}
	eq := strings.IndexByte(s, '=')
	if eq <= 0 {
		return ""
	}
	key := strings.TrimSpace(s[:eq])
	if key == "" || strings.ContainsAny(key, " \t") {
		return ""
	}
	return key
}
