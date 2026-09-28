package env

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode"
)

const maxValueBytes = 64 << 10

// Parse reads KEY=VALUE dotenv syntax into a map. It does not touch the
// process environment. Process lookup is used only for ${VAR} expansion
// when the name is not defined in the same file.
func Parse(data []byte) (map[string]string, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")

	raw := make(map[string]string)
	expandable := make(map[string]bool)
	i := 0
	for i < len(lines) {
		lineNo := i + 1
		s := strings.TrimSpace(lines[i])
		if s == "" || strings.HasPrefix(s, "#") {
			i++
			continue
		}
		key, val, doExpand, next, err := readAssignment(lines, i)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		raw[key] = val
		expandable[key] = doExpand
		i = next
	}

	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	for n := 0; n < 32; n++ {
		changed := false
		for k, ok := range expandable {
			if !ok {
				continue
			}
			next := expand(out[k], out)
			if len(next) > maxValueBytes {
				next = next[:maxValueBytes]
			}
			if next != out[k] {
				out[k] = next
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return out, nil
}

func readAssignment(lines []string, i int) (key, value string, expand bool, next int, err error) {
	s := strings.TrimSpace(lines[i])
	if strings.HasPrefix(s, "export") && (len(s) == 6 || unicode.IsSpace(rune(s[6]))) {
		s = strings.TrimSpace(s[6:])
	}
	eq := strings.IndexByte(s, '=')
	if eq <= 0 {
		return "", "", false, i, fmt.Errorf("expected KEY=VALUE")
	}
	key = strings.TrimSpace(s[:eq])
	if key == "" || strings.ContainsAny(key, " \t") {
		return "", "", false, i, fmt.Errorf("invalid key %q", key)
	}
	rest := s[eq+1:]
	value, expand, consumed, err := readValue(rest, lines, i)
	if err != nil {
		return "", "", false, i, err
	}
	return key, value, expand, consumed, nil
}

func readValue(rest string, lines []string, i int) (value string, expand bool, next int, err error) {
	trimmedLeft := strings.TrimLeft(rest, " \t")
	if trimmedLeft == "" {
		return "", true, i + 1, nil
	}
	switch trimmedLeft[0] {
	case '"':
		body, endLine, err := readQuoted(lines, i, rest, '"', true)
		return body, true, endLine, err
	case '\'':
		body, endLine, err := readQuoted(lines, i, rest, '\'', false)
		return body, false, endLine, err
	default:
		v := rest
		if idx := indexUnquotedComment(v); idx >= 0 {
			v = v[:idx]
		}
		return strings.TrimSpace(v), true, i + 1, nil
	}
}

func indexUnquotedComment(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '#' && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t') {
			return i
		}
	}
	return -1
}

func readQuoted(lines []string, start int, firstRest string, quote byte, unescape bool) (string, int, error) {
	// firstRest still has leading spaces and the opening quote somewhere.
	idx := strings.IndexByte(firstRest, quote)
	if idx < 0 {
		return "", start, fmt.Errorf("missing opening quote")
	}
	var b strings.Builder
	line := firstRest[idx+1:]
	i := start
	for {
		closed, rest, err := scanQuotedLine(line, quote, unescape, &b)
		if err != nil {
			return "", start, err
		}
		if closed {
			if b.Len() > maxValueBytes {
				return "", start, fmt.Errorf("value too large")
			}
			if strings.TrimSpace(rest) != "" && !strings.HasPrefix(strings.TrimSpace(rest), "#") {
				return "", start, fmt.Errorf("trailing characters after quoted value")
			}
			return b.String(), i + 1, nil
		}
		i++
		if i >= len(lines) {
			return "", start, fmt.Errorf("unterminated quote")
		}
		if b.Len() > maxValueBytes {
			return "", start, fmt.Errorf("value too large")
		}
		b.WriteByte('\n')
		line = lines[i]
	}
}

func scanQuotedLine(line string, quote byte, unescape bool, b *strings.Builder) (closed bool, rest string, err error) {
	for i := 0; i < len(line); i++ {
		c := line[i]
		if unescape && c == '\\' && i+1 < len(line) {
			n := line[i+1]
			switch n {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '\\', '"', '\'':
				b.WriteByte(n)
			case '$':
				b.WriteByte('$')
			default:
				b.WriteByte('\\')
				b.WriteByte(n)
			}
			i++
			continue
		}
		if c == quote {
			return true, line[i+1:], nil
		}
		b.WriteByte(c)
	}
	return false, "", nil
}

func expand(s string, file map[string]string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if b.Len() >= maxValueBytes {
			return b.String()[:maxValueBytes]
		}
		if s[i] != '$' {
			b.WriteByte(s[i])
			i++
			continue
		}
		if i+1 < len(s) && s[i+1] == '{' {
			end := strings.IndexByte(s[i+2:], '}')
			if end < 0 {
				b.WriteByte('$')
				i++
				continue
			}
			name := s[i+2 : i+2+end]
			if !writeCapped(&b, lookupExpand(name, file)) {
				return b.String()
			}
			i = i + 3 + end
			continue
		}
		j := i + 1
		if j < len(s) && (s[j] == '_' || isIdentStart(s[j])) {
			j++
			for j < len(s) && isIdentPart(s[j]) {
				j++
			}
			if !writeCapped(&b, lookupExpand(s[i+1:j], file)) {
				return b.String()
			}
			i = j
			continue
		}
		b.WriteByte('$')
		i++
	}
	if b.Len() > maxValueBytes {
		return b.String()[:maxValueBytes]
	}
	return b.String()
}

func writeCapped(b *strings.Builder, s string) bool {
	remain := maxValueBytes - b.Len()
	if remain <= 0 {
		return false
	}
	if len(s) > remain {
		b.WriteString(s[:remain])
		return false
	}
	b.WriteString(s)
	return true
}

func lookupExpand(name string, file map[string]string) string {
	if name == "" {
		return ""
	}
	if v, ok := file[name]; ok {
		return v
	}
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return ""
}

func isIdentStart(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || c >= '0' && c <= '9' || c == '_'
}
