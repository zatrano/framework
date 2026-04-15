package config

import "strings"

// MaskConnectionURL hides passwords in postgres/redis-style URLs for logs and CLI output.
func MaskConnectionURL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(empty)"
	}
	if i := strings.Index(s, "@"); i > 0 {
		head := s[:i]
		if j := strings.LastIndex(head, ":"); j > 0 && strings.Contains(head, "://") {
			return head[:j+1] + "***@" + s[i+1:]
		}
	}
	return "(set)"
}

// MaskSecret returns whether a secret is configured without revealing it.
func MaskSecret(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(empty)"
	}
	return "(set)"
}

