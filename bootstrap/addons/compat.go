package addons

import (
	"strconv"
	"strings"
)

// MeetsFrameworkMin reports whether have (kernel VERSION) satisfies min.
// Empty min is unspecified and always meets.
func MeetsFrameworkMin(have, min string) bool {
	min = strings.TrimSpace(strings.TrimPrefix(min, "v"))
	if min == "" {
		return true
	}
	have = strings.TrimSpace(strings.TrimPrefix(have, "v"))
	if have == "" {
		return false
	}
	return compareSemver(have, min) >= 0
}

func compareSemver(a, b string) int {
	as := semverParts(a)
	bs := semverParts(b)
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func semverParts(v string) []int {
	v = strings.Split(v, "-")[0]
	v = strings.Split(v, "+")[0]
	chunks := strings.Split(v, ".")
	out := make([]int, 0, len(chunks))
	for _, c := range chunks {
		n, err := strconv.Atoi(c)
		if err != nil {
			out = append(out, 0)
			continue
		}
		out = append(out, n)
	}
	return out
}
