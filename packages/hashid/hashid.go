package hashid

import (
	"fmt"
	"strings"
)

const defaultAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

// Hasher encodes integers into opaque reversible strings.
type Hasher struct {
	salt      string
	alphabet  []rune
	separator rune
	minLength int
}

// New creates a hasher.
func New(salt string, minLength ...int) *Hasher {
	minLen := 0
	if len(minLength) > 0 && minLength[0] > 0 {
		minLen = minLength[0]
	}
	shuffled := shuffle(defaultAlphabet, salt+"zatrano")
	return &Hasher{
		salt:      salt,
		separator: '-',
		alphabet:  []rune(shuffled),
		minLength: minLen,
	}
}

// Encode encodes one or more non-negative integers.
func (h *Hasher) Encode(nums ...int64) (string, error) {
	if len(nums) == 0 {
		return "", fmt.Errorf("hashid: no numbers")
	}
	parts := make([]string, 0, len(nums))
	for _, n := range nums {
		if n < 0 {
			return "", fmt.Errorf("hashid: negative numbers are not supported")
		}
		parts = append(parts, toBase(n, h.alphabet))
	}
	body := strings.Join(parts, string(h.separator))
	check := h.alphabet[checksum(body+h.salt)%len(h.alphabet)]
	out := string(check) + body
	i := 0
	for len(out) < h.minLength {
		out = string(h.alphabet[(i+int(check))%len(h.alphabet)]) + out
		i++
	}
	return out, nil
}

// Decode restores numbers from a hash produced by Encode.
func (h *Hasher) Decode(hash string) ([]int64, error) {
	if hash == "" {
		return nil, fmt.Errorf("hashid: empty hash")
	}
	maxPad := h.minLength
	if maxPad > len(hash)-1 {
		maxPad = len(hash) - 1
	}
	if maxPad < 0 {
		maxPad = 0
	}
	for pad := 0; pad <= maxPad; pad++ {
		core := hash[pad:]
		if len(core) < 2 {
			continue
		}
		check := []rune(core)[0]
		body := string([]rune(core)[1:])
		if check != h.alphabet[checksum(body+h.salt)%len(h.alphabet)] {
			continue
		}
		parts := strings.Split(body, string(h.separator))
		nums := make([]int64, 0, len(parts))
		ok := true
		for _, part := range parts {
			if part == "" {
				ok = false
				break
			}
			n, err := fromBase(part, h.alphabet)
			if err != nil {
				ok = false
				break
			}
			nums = append(nums, n)
		}
		if !ok || len(nums) == 0 {
			continue
		}
		encoded, err := h.Encode(nums...)
		if err == nil && encoded == hash {
			return nums, nil
		}
	}
	return nil, fmt.Errorf("hashid: invalid hash")
}

func checksum(s string) int {
	sum := 0
	for _, r := range s {
		sum += int(r)
	}
	return sum
}

func toBase(n int64, alphabet []rune) string {
	if n == 0 {
		return string(alphabet[0])
	}
	base := int64(len(alphabet))
	var buf []rune
	for n > 0 {
		buf = append([]rune{alphabet[n%base]}, buf...)
		n /= base
	}
	return string(buf)
}

func fromBase(s string, alphabet []rune) (int64, error) {
	base := int64(len(alphabet))
	var n int64
	for _, r := range s {
		idx := indexRune(alphabet, r)
		if idx < 0 {
			return 0, fmt.Errorf("hashid: invalid character")
		}
		n = n*base + int64(idx)
	}
	return n, nil
}

func shuffle(alphabet, salt string) string {
	runes := []rune(alphabet)
	if salt == "" {
		return alphabet
	}
	for i, v, p := len(runes)-1, 0, 0; i > 0; i-- {
		v += int(salt[p])
		j := (int(salt[p]) + p + v) % i
		runes[j], runes[i] = runes[i], runes[j]
		p = (p + 1) % len(salt)
	}
	return string(runes)
}

func indexRune(alphabet []rune, r rune) int {
	for i, a := range alphabet {
		if a == r {
			return i
		}
	}
	return -1
}
