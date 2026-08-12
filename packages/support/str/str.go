package str

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/zatrano/framework/packages/support/uuid"
)

// Of wraps a string for fluent operations.
type Of string

// New creates a fluent string wrapper.
func New(value string) Of {
	return Of(value)
}

// String returns the underlying string.
func (s Of) String() string {
	return string(s)
}

// Lower returns lowercase.
func Lower(value string) string { return strings.ToLower(value) }

// Upper returns uppercase.
func Upper(value string) string { return strings.ToUpper(value) }

// Title converts to title case.
func Title(value string) string {
	parts := strings.Fields(value)
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

// Camel converts snake/kebab to camelCase.
func Camel(value string) string {
	studly := Studly(value)
	if studly == "" {
		return ""
	}
	runes := []rune(studly)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// Studly converts snake/kebab to StudlyCase.
func Studly(value string) string {
	value = strings.ReplaceAll(value, "-", "_")
	parts := strings.Split(value, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, "")
}

// Snake converts Studly/Camel to snake_case.
func Snake(value string) string {
	var b strings.Builder
	for i, r := range value {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		if r == '-' || r == ' ' {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Kebab converts to kebab-case.
func Kebab(value string) string {
	return strings.ReplaceAll(Snake(value), "_", "-")
}

// Slug creates a URL slug.
func Slug(value string, separator ...string) string {
	sep := "-"
	if len(separator) > 0 && separator[0] != "" {
		sep = separator[0]
	}
	value = strings.ToLower(strings.TrimSpace(value))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	value = re.ReplaceAllString(value, sep)
	return strings.Trim(value, sep)
}

// Contains reports whether value contains needle.
func Contains(value, needle string) bool {
	return strings.Contains(value, needle)
}

// ContainsAll reports whether value contains all needles.
func ContainsAll(value string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}

// ContainsAny reports whether value contains any needle.
func ContainsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

// Headline converts a string to a headline (Title Case words).
func Headline(value string) string {
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.ReplaceAll(value, "_", " ")
	return Title(value)
}

// StartsWith reports whether value starts with prefix.
func StartsWith(value, prefix string) bool {
	return strings.HasPrefix(value, prefix)
}

// EndsWith reports whether value ends with suffix.
func EndsWith(value, suffix string) bool {
	return strings.HasSuffix(value, suffix)
}

// StartsWithAny reports whether value starts with any of the prefixes.
func StartsWithAny(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if StartsWith(value, prefix) {
			return true
		}
	}
	return false
}

// EndsWithAny reports whether value ends with any of the suffixes.
func EndsWithAny(value string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if EndsWith(value, suffix) {
			return true
		}
	}
	return false
}

// Position returns the rune index of the first occurrence of needle, or -1.
func Position(value, needle string) int {
	if needle == "" {
		return 0
	}
	i := strings.Index(value, needle)
	if i < 0 {
		return -1
	}
	return utf8.RuneCountInString(value[:i])
}

// LastPosition returns the rune index of the last occurrence of needle, or -1.
func LastPosition(value, needle string) int {
	if needle == "" {
		return utf8.RuneCountInString(value)
	}
	i := strings.LastIndex(value, needle)
	if i < 0 {
		return -1
	}
	return utf8.RuneCountInString(value[:i])
}

// Limit truncates a string with optional end.
func Limit(value string, limit int, end ...string) string {
	suffix := "..."
	if len(end) > 0 {
		suffix = end[0]
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= 0 {
		return suffix
	}
	return string(runes[:limit]) + suffix
}

// Excerpt truncates on a word boundary when possible.
func Excerpt(value string, limit int, end ...string) string {
	suffix := "..."
	if len(end) > 0 {
		suffix = end[0]
	}
	runes := []rune(strings.TrimSpace(value))
	if limit <= 0 || len(runes) <= limit {
		return string(runes)
	}
	cut := runes[:limit]
	if i := strings.LastIndex(string(cut), " "); i > 0 {
		cut = []rune(string(cut)[:i])
	}
	return strings.TrimSpace(string(cut)) + suffix
}

// Mask hides the middle of a string, keeping start/end visible characters.
func Mask(value string, start, end int, mask ...string) string {
	repl := "*"
	if len(mask) > 0 && mask[0] != "" {
		repl = mask[0]
	}
	runes := []rune(value)
	n := len(runes)
	if n == 0 {
		return ""
	}
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start+end >= n {
		return strings.Repeat(repl, n)
	}
	hidden := n - start - end
	return string(runes[:start]) + strings.Repeat(repl, hidden) + string(runes[n-end:])
}

// Random returns a random hex string of n bytes.
func Random(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

// After returns the remainder after the first search occurrence.
func After(value, search string) string {
	if search == "" {
		return value
	}
	idx := strings.Index(value, search)
	if idx < 0 {
		return value
	}
	return value[idx+len(search):]
}

// Before returns the content before the first search occurrence.
func Before(value, search string) string {
	if search == "" {
		return value
	}
	idx := strings.Index(value, search)
	if idx < 0 {
		return value
	}
	return value[:idx]
}

// Between returns the content between the first from and the last to.
func Between(value, from, to string) string {
	if from == "" || to == "" {
		return value
	}
	return BeforeLast(After(value, from), to)
}

// BetweenFirst returns the smallest portion between the first from and first to.
func BetweenFirst(value, from, to string) string {
	if from == "" || to == "" {
		return value
	}
	return Before(After(value, from), to)
}

// AfterLast returns the remainder after the last search occurrence.
func AfterLast(value, search string) string {
	if search == "" {
		return value
	}
	idx := strings.LastIndex(value, search)
	if idx < 0 {
		return value
	}
	return value[idx+len(search):]
}

// BeforeLast returns the content before the last search occurrence.
func BeforeLast(value, search string) string {
	if search == "" {
		return value
	}
	idx := strings.LastIndex(value, search)
	if idx < 0 {
		return value
	}
	return value[:idx]
}

// Chop removes the last character (rune) from value.
func Chop(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

// Take returns the first n runes. Negative n takes from the end (see TakeLast).
func Take(value string, n int) string {
	if n < 0 {
		return TakeLast(value, -n)
	}
	runes := []rune(value)
	if n == 0 {
		return ""
	}
	if n >= len(runes) {
		return value
	}
	return string(runes[:n])
}

// TakeLast returns the last n runes.
func TakeLast(value string, n int) string {
	runes := []rune(value)
	if n <= 0 {
		return ""
	}
	if n >= len(runes) {
		return value
	}
	return string(runes[len(runes)-n:])
}

// Unwrap removes before from the start and after from the end when present.
// If after is omitted, before is used for both sides.
func Unwrap(value, before string, after ...string) string {
	end := before
	if len(after) > 0 {
		end = after[0]
	}
	if before != "" && strings.HasPrefix(value, before) {
		value = value[len(before):]
	}
	if end != "" && strings.HasSuffix(value, end) {
		value = value[:len(value)-len(end)]
	}
	return value
}

// Substr returns a rune-based substring starting at start with optional length.
// Negative start counts from the end.
func Substr(value string, start int, length ...int) string {
	runes := []rune(value)
	n := len(runes)
	if n == 0 {
		return ""
	}
	if start < 0 {
		start = n + start
	}
	if start < 0 {
		start = 0
	}
	if start >= n {
		return ""
	}
	end := n
	if len(length) > 0 {
		if length[0] < 0 {
			return ""
		}
		end = start + length[0]
		if end > n {
			end = n
		}
	}
	return string(runes[start:end])
}

// Replace replaces all occurrences.
func Replace(value, search, replacement string) string {
	return strings.ReplaceAll(value, search, replacement)
}

// ReplaceFirst replaces the first occurrence of search.
func ReplaceFirst(value, search, replacement string) string {
	if search == "" {
		return value
	}
	idx := strings.Index(value, search)
	if idx < 0 {
		return value
	}
	return value[:idx] + replacement + value[idx+len(search):]
}

// ReplaceLast replaces the last occurrence of search.
func ReplaceLast(value, search, replacement string) string {
	if search == "" {
		return value
	}
	idx := strings.LastIndex(value, search)
	if idx < 0 {
		return value
	}
	return value[:idx] + replacement + value[idx+len(search):]
}

// ReplaceStart replaces search only when it is a prefix of value.
func ReplaceStart(value, search, replacement string) string {
	if search == "" || !strings.HasPrefix(value, search) {
		return value
	}
	return replacement + value[len(search):]
}

// ReplaceEnd replaces search only when it is a suffix of value.
func ReplaceEnd(value, search, replacement string) string {
	if search == "" || !strings.HasSuffix(value, search) {
		return value
	}
	return value[:len(value)-len(search)] + replacement
}

// ReplaceArray replaces successive occurrences of search with replacements in order.
func ReplaceArray(value, search string, replacements []string) string {
	if search == "" {
		return value
	}
	for _, repl := range replacements {
		idx := strings.Index(value, search)
		if idx < 0 {
			break
		}
		value = value[:idx] + repl + value[idx+len(search):]
	}
	return value
}

// SubstrReplace replaces a rune slice of value starting at start (optional length; omit to replace through end).
func SubstrReplace(value, replacement string, start int, length ...int) string {
	runes := []rune(value)
	n := len(runes)
	if start < 0 {
		start = n + start
		if start < 0 {
			start = 0
		}
	}
	if start > n {
		start = n
	}
	end := n
	if len(length) > 0 {
		l := length[0]
		if l < 0 {
			end = n + l
			if end < start {
				end = start
			}
		} else {
			end = start + l
			if end > n {
				end = n
			}
		}
	}
	return string(runes[:start]) + replacement + string(runes[end:])
}

// UcFirst uppercases the first character.
func UcFirst(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// LcFirst lowercases the first character.
func LcFirst(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// Is reports whether value matches a pattern with * wildcards.
func Is(pattern, value string) bool {
	if pattern == value {
		return true
	}
	re := regexp.MustCompile("^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, `.*`) + "$")
	return re.MatchString(value)
}

// Match returns the first regex match (or first capture group when present).
func Match(pattern, value string) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}
	m := re.FindStringSubmatch(value)
	if len(m) == 0 {
		return ""
	}
	if len(m) > 1 {
		return m[1]
	}
	return m[0]
}

// IsMatch reports whether value matches the regex pattern.
func IsMatch(pattern, value string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(value)
}

// MatchAll returns all regex matches (or first capture group of each match).
func MatchAll(pattern, value string) []string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	all := re.FindAllStringSubmatch(value, -1)
	out := make([]string, 0, len(all))
	for _, m := range all {
		if len(m) > 1 {
			out = append(out, m[1])
			continue
		}
		if len(m) > 0 {
			out = append(out, m[0])
		}
	}
	return out
}

// Numbers keeps only digit runes from value.
func Numbers(value string) string {
	var b strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Length returns rune count.
func Length(value string) int {
	return len([]rune(value))
}

// Repeat returns value repeated n times (empty when n <= 0).
func Repeat(value string, n int) string {
	if n <= 0 || value == "" {
		return ""
	}
	return strings.Repeat(value, n)
}

// IsEmpty reports whether value has zero length.
func IsEmpty(value string) bool {
	return value == ""
}

// IsNotEmpty reports whether value has non-zero length.
func IsNotEmpty(value string) bool {
	return value != ""
}

// IsBlank reports whether value is empty or only whitespace.
func IsBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}

// IsNotBlank reports whether value contains non-whitespace characters.
func IsNotBlank(value string) bool {
	return !IsBlank(value)
}

// IsUpper reports whether value equals its uppercase form.
func IsUpper(value string) bool {
	return strings.ToUpper(value) == value
}

// IsLower reports whether value equals its lowercase form.
func IsLower(value string) bool {
	return strings.ToLower(value) == value
}

// IsAlpha reports whether value contains only letters.
func IsAlpha(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// IsAlphaNumeric reports whether value contains only letters and digits.
func IsAlphaNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsDigit reports whether value contains only decimal digits.
func IsDigit(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsHex reports whether value contains only hexadecimal digits.
func IsHex(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// IsUuid reports whether value is a valid UUID.
func IsUuid(value string) bool {
	return uuid.IsValid(strings.TrimSpace(value))
}

var ulidPattern = regexp.MustCompile(`(?i)^[0-9A-HJKMNP-TV-Z]{26}$`)

// IsUlid reports whether value is a valid ULID.
func IsUlid(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) == 26 && ulidPattern.MatchString(value)
}

// IsJson reports whether value is valid JSON.
func IsJson(value string) bool {
	return json.Valid([]byte(strings.TrimSpace(value)))
}

// IsUrl reports whether value is an absolute URL with scheme and host.
func IsUrl(value string) bool {
	u, err := url.ParseRequestURI(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// IsEmail reports whether value looks like an email address.
func IsEmail(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && emailPattern.MatchString(value)
}

// IsIp reports whether value is a valid IPv4 or IPv6 address.
func IsIp(value string) bool {
	return net.ParseIP(strings.TrimSpace(value)) != nil
}

// IsMac reports whether value is a valid MAC address.
func IsMac(value string) bool {
	_, err := net.ParseMAC(strings.TrimSpace(value))
	return err == nil
}

var semverPattern = regexp.MustCompile(`(?i)^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

// IsSemver reports whether value is a valid semantic version (optional leading v).
func IsSemver(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && semverPattern.MatchString(value)
}

// PadLeft left-pads value to length with pad (default space).
func PadLeft(value string, length int, pad ...string) string {
	return padString(value, length, true, pad...)
}

// PadRight right-pads value to length with pad (default space).
func PadRight(value string, length int, pad ...string) string {
	return padString(value, length, false, pad...)
}

// PadBoth pads value on both sides to reach length.
func PadBoth(value string, length int, pad ...string) string {
	runes := []rune(value)
	if len(runes) >= length {
		return value
	}
	need := length - len(runes)
	left := need / 2
	return PadRight(PadLeft(value, len(runes)+left, pad...), length, pad...)
}

// CharAt returns the rune at index (negative indexes count from the end).
func CharAt(value string, index int) string {
	runes := []rune(value)
	n := len(runes)
	if n == 0 {
		return ""
	}
	if index < 0 {
		index = n + index
	}
	if index < 0 || index >= n {
		return ""
	}
	return string(runes[index])
}

// SubstrCount returns the number of non-overlapping needle occurrences.
func SubstrCount(value, needle string) int {
	if needle == "" {
		return 0
	}
	return strings.Count(value, needle)
}

// WordWrap wraps value to lines of at most width runes (word-aware).
func WordWrap(value string, width int) string {
	if width <= 0 {
		return value
	}
	words := strings.Fields(value)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	var line []rune
	for _, word := range words {
		w := []rune(word)
		if len(line) == 0 {
			line = append([]rune{}, w...)
			continue
		}
		if len(line)+1+len(w) <= width {
			line = append(line, ' ')
			line = append(line, w...)
			continue
		}
		lines = append(lines, string(line))
		line = append([]rune{}, w...)
	}
	if len(line) > 0 {
		lines = append(lines, string(line))
	}
	return strings.Join(lines, "\n")
}

func padString(value string, length int, left bool, pad ...string) string {
	fill := " "
	if len(pad) > 0 && pad[0] != "" {
		fill = pad[0]
	}
	runes := []rune(value)
	if len(runes) >= length {
		return value
	}
	need := length - len(runes)
	var b strings.Builder
	fillRunes := []rune(fill)
	if len(fillRunes) == 0 {
		fillRunes = []rune(" ")
	}
	for i := 0; i < need; i++ {
		b.WriteRune(fillRunes[i%len(fillRunes)])
	}
	if left {
		return b.String() + value
	}
	return value + b.String()
}

// Trim removes characters from both ends. With no characters, trims Unicode spaces.
func Trim(value string, characters ...string) string {
	if len(characters) == 0 || characters[0] == "" {
		return strings.TrimSpace(value)
	}
	return strings.Trim(value, characters[0])
}

// LTrim removes characters from the start. With no characters, trims Unicode spaces.
func LTrim(value string, characters ...string) string {
	if len(characters) == 0 || characters[0] == "" {
		return strings.TrimLeftFunc(value, unicode.IsSpace)
	}
	return strings.TrimLeft(value, characters[0])
}

// RTrim removes characters from the end. With no characters, trims Unicode spaces.
func RTrim(value string, characters ...string) string {
	if len(characters) == 0 || characters[0] == "" {
		return strings.TrimRightFunc(value, unicode.IsSpace)
	}
	return strings.TrimRight(value, characters[0])
}

// Indent prefixes each line with pad repeated amount times (default pad is a space).
func Indent(value string, amount int, pad ...string) string {
	unit := " "
	if len(pad) > 0 && pad[0] != "" {
		unit = pad[0]
	}
	if amount < 0 {
		amount = 0
	}
	prefix := strings.Repeat(unit, amount)
	if prefix == "" {
		return value
	}
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// Dedent removes the common leading whitespace from each non-empty line.
func Dedent(value string) string {
	lines := strings.Split(value, "\n")
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := 0
		for _, r := range line {
			if r == ' ' || r == '\t' {
				indent++
				continue
			}
			break
		}
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return value
	}
	for i, line := range lines {
		runes := []rune(line)
		if len(runes) >= minIndent {
			lines[i] = string(runes[minIndent:])
		}
	}
	return strings.Join(lines, "\n")
}

// Explode splits value by delimiter. Optional limit uses SplitN semantics.
func Explode(delimiter, value string, limit ...int) []string {
	if len(limit) > 0 {
		return strings.SplitN(value, delimiter, limit[0])
	}
	return strings.Split(value, delimiter)
}

// Join concatenates parts with delimiter.
func Join(parts []string, delimiter string) string {
	return strings.Join(parts, delimiter)
}

// Lines splits value into lines (normalizes CRLF to LF first).
func Lines(value string) []string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.Split(normalized, "\n")
}

// Squish collapses consecutive whitespace into a single space and trims.
func Squish(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// WordCount returns the number of whitespace-separated words.
func WordCount(value string) int {
	fields := strings.Fields(strings.TrimSpace(value))
	return len(fields)
}

// Finish ensures value ends with suffix.
func Finish(value, suffix string) string {
	if suffix == "" || strings.HasSuffix(value, suffix) {
		return value
	}
	return value + suffix
}

// Start ensures value begins with prefix.
func Start(value, prefix string) string {
	if prefix == "" || strings.HasPrefix(value, prefix) {
		return value
	}
	return prefix + value
}

// ChopStart removes the first matching prefix from value.
func ChopStart(value string, needles ...string) string {
	for _, needle := range needles {
		if needle != "" && strings.HasPrefix(value, needle) {
			return value[len(needle):]
		}
	}
	return value
}

// ChopEnd removes the first matching suffix from value.
func ChopEnd(value string, needles ...string) string {
	for _, needle := range needles {
		if needle != "" && strings.HasSuffix(value, needle) {
			return value[:len(value)-len(needle)]
		}
	}
	return value
}

// ToBase64 encodes value as standard base64.
func ToBase64(value string) string {
	return base64.StdEncoding.EncodeToString([]byte(value))
}

// FromBase64 decodes a standard base64 string (optional fallback on error).
func FromBase64(value string, fallback ...string) string {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	return string(raw)
}

// Reverse reverses value by Unicode code points.
func Reverse(value string) string {
	runes := []rune(value)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Wrap surrounds value with before and after (after defaults to before).
func Wrap(value, before string, after ...string) string {
	end := before
	if len(after) > 0 {
		end = after[0]
	}
	return before + value + end
}

// Words keeps at most n whitespace-separated words, appending an optional end (default "...").
func Words(value string, n int, end ...string) string {
	suffix := "..."
	if len(end) > 0 {
		suffix = end[0]
	}
	fields := strings.Fields(strings.TrimSpace(value))
	if n < 0 {
		n = 0
	}
	if len(fields) <= n {
		return strings.Join(fields, " ")
	}
	return strings.Join(fields[:n], " ") + suffix
}

// IsAscii reports whether value contains only ASCII bytes.
func IsAscii(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// Ascii transliterates common Latin/Turkish letters to ASCII and drops other non-ASCII.
func Ascii(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if r <= unicode.MaxASCII {
			b.WriteRune(r)
			continue
		}
		if s, ok := asciiFold[r]; ok {
			b.WriteString(s)
		}
	}
	return b.String()
}

var asciiFold = map[rune]string{
	'À': "A", 'Á': "A", 'Â': "A", 'Ã': "A", 'Ä': "A", 'Å': "A", 'Æ': "AE",
	'Ç': "C", 'È': "E", 'É': "E", 'Ê': "E", 'Ë': "E", 'Ì': "I", 'Í': "I",
	'Î': "I", 'Ï': "I", 'Ð': "D", 'Ñ': "N", 'Ò': "O", 'Ó': "O", 'Ô': "O",
	'Õ': "O", 'Ö': "O", 'Ø': "O", 'Ù': "U", 'Ú': "U", 'Û': "U", 'Ü': "U",
	'Ý': "Y", 'Þ': "TH", 'ß': "ss",
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'æ': "ae",
	'ç': "c", 'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ì': "i", 'í': "i",
	'î': "i", 'ï': "i", 'ð': "d", 'ñ': "n", 'ò': "o", 'ó': "o", 'ô': "o",
	'õ': "o", 'ö': "o", 'ø': "o", 'ù': "u", 'ú': "u", 'û': "u", 'ü': "u",
	'ý': "y", 'þ': "th", 'ÿ': "y",
	'Ğ': "G", 'ğ': "g", 'İ': "I", 'ı': "i", 'Ş': "S", 'ş': "s",
}

// Swap replaces multiple search=>replace pairs.
func Swap(value string, replacements map[string]string) string {
	for search, replace := range replacements {
		value = strings.ReplaceAll(value, search, replace)
	}
	return value
}

// Remove removes all occurrences of the given needles.
func Remove(value string, needles ...string) string {
	for _, needle := range needles {
		value = strings.ReplaceAll(value, needle, "")
	}
	return value
}

// Lower fluent.
func (s Of) Lower() Of { return Of(Lower(string(s))) }

// Upper fluent.
func (s Of) Upper() Of { return Of(Upper(string(s))) }

// Snake fluent.
func (s Of) Snake() Of { return Of(Snake(string(s))) }

// Slug fluent.
func (s Of) Slug() Of { return Of(Slug(string(s))) }

// Limit fluent.
func (s Of) Limit(n int, end ...string) Of { return Of(Limit(string(s), n, end...)) }
