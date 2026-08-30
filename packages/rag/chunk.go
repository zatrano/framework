package rag

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// TextChunker splits on rune length with optional overlap.
type TextChunker struct {
	Size     int // max runes per chunk (default 800)
	Overlap  int // overlapping runes between chunks (default 100)
	MinChars int // drop tiny trailing scraps (default 40)
}

// Split implements Chunker.
func (c TextChunker) Split(doc Document) []Chunk {
	size := c.Size
	if size <= 0 {
		size = 800
	}
	overlap := c.Overlap
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 4
	}
	minChars := c.MinChars
	if minChars <= 0 {
		minChars = 40
	}
	text := strings.TrimSpace(doc.Text)
	if text == "" {
		return nil
	}
	runes := []rune(text)
	meta := cloneMeta(doc.Metadata)
	var out []Chunk
	for start := 0; start < len(runes); {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		piece := strings.TrimSpace(string(runes[start:end]))
		if utf8.RuneCountInString(piece) >= minChars || start == 0 {
			idx := len(out)
			out = append(out, Chunk{
				ID:         fmt.Sprintf("%s#%d", firstNonEmpty(doc.ID, "doc"), idx),
				DocumentID: firstNonEmpty(doc.ID, "doc"),
				Text:       piece,
				Index:      idx,
				Metadata:   meta,
			})
		}
		if end >= len(runes) {
			break
		}
		next := end - overlap
		if next <= start {
			next = start + 1
		}
		start = next
	}
	return out
}

func cloneMeta(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
