package str

import "strings"

var irregularPlurals = map[string]string{
	"child":    "children",
	"person":   "people",
	"man":      "men",
	"woman":    "women",
	"mouse":    "mice",
	"goose":    "geese",
	"tooth":    "teeth",
	"foot":     "feet",
	"ox":       "oxen",
	"leaf":     "leaves",
	"life":     "lives",
	"wife":     "wives",
	"knife":    "knives",
	"potato":   "potatoes",
	"tomato":   "tomatoes",
	"cactus":   "cacti",
	"focus":    "foci",
	"analysis": "analyses",
	"index":    "indices",
	"matrix":   "matrices",
	"quiz":     "quizzes",
	"bus":      "buses",
}

var irregularSingulars = map[string]string{}

func init() {
	for singular, plural := range irregularPlurals {
		irregularSingulars[plural] = singular
	}
}

// Plural returns the plural form of word. If count is 1, returns singular.
func Plural(word string, count ...int) string {
	if len(count) > 0 && count[0] == 1 {
		return word
	}
	lower := strings.ToLower(word)
	if plural, ok := irregularPlurals[lower]; ok {
		return matchCase(word, plural)
	}
	if strings.HasSuffix(lower, "y") && len(lower) > 1 && !isVowel(rune(lower[len(lower)-2])) {
		return matchCase(word, word[:len(word)-1]+"ies")
	}
	for _, suf := range []string{"s", "ss", "sh", "ch", "x", "z", "o"} {
		if strings.HasSuffix(lower, suf) {
			if suf == "s" || suf == "ss" || suf == "sh" || suf == "ch" || suf == "x" || suf == "z" {
				return word + "es"
			}
		}
	}
	if strings.HasSuffix(lower, "fe") {
		return word[:len(word)-2] + "ves"
	}
	if strings.HasSuffix(lower, "f") {
		return word[:len(word)-1] + "ves"
	}
	return word + "s"
}

// Singular returns a best-effort singular form.
func Singular(word string) string {
	lower := strings.ToLower(word)
	if singular, ok := irregularSingulars[lower]; ok {
		return matchCase(word, singular)
	}
	if strings.HasSuffix(lower, "ies") && len(word) > 3 {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(lower, "ves") && len(word) > 3 {
		return word[:len(word)-3] + "f"
	}
	for _, suf := range []string{"xes", "ches", "shes", "sses", "zes"} {
		if strings.HasSuffix(lower, suf) {
			return word[:len(word)-2]
		}
	}
	if strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss") && len(word) > 1 {
		return word[:len(word)-1]
	}
	return word
}

// PluralStudly is an alias used by demos.
func Pluralize(word string, count ...int) string { return Plural(word, count...) }

func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}

func matchCase(sample, value string) string {
	if sample == "" {
		return value
	}
	if sample == strings.ToUpper(sample) {
		return strings.ToUpper(value)
	}
	if sample[0] >= 'A' && sample[0] <= 'Z' {
		runes := []rune(value)
		if len(runes) == 0 {
			return value
		}
		return strings.ToUpper(string(runes[0])) + string(runes[1:])
	}
	return value
}
