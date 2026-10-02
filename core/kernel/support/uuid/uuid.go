package uuid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// New generates a random UUID v4 string.
func New() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("uuid: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return Format(b)
}

// Format renders a 16-byte UUID as a canonical string.
func Format(b [16]byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Parse parses a UUID string into 16 bytes.
func Parse(s string) ([16]byte, error) {
	var out [16]byte
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 32 {
		return out, fmt.Errorf("uuid: invalid length")
	}
	raw, err := hex.DecodeString(s)
	if err != nil {
		return out, fmt.Errorf("uuid: %w", err)
	}
	copy(out[:], raw)
	return out, nil
}

// IsValid reports whether s is a valid UUID (with or without hyphens).
func IsValid(s string) bool {
	_, err := Parse(s)
	return err == nil
}

// IsV4 reports whether s is a valid UUID version 4.
func IsV4(s string) bool {
	b, err := Parse(s)
	if err != nil {
		return false
	}
	version := (b[6] >> 4) & 0x0f
	variant := (b[8] >> 6) & 0x03
	return version == 4 && variant == 2
}
