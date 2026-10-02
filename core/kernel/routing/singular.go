package routing

import "strings"

var irregularSingulars = map[string]string{
	"children": "child",
	"people":   "person",
	"men":      "man",
	"women":    "woman",
	"mice":     "mouse",
	"geese":    "goose",
	"teeth":    "tooth",
	"feet":     "foot",
	"oxen":     "ox",
	"leaves":   "leaf",
	"lives":    "life",
	"wives":    "wife",
	"knives":   "knife",
	"potatoes": "potato",
	"tomatoes": "tomato",
	"cacti":    "cactus",
	"foci":     "focus",
	"analyses": "analysis",
	"indices":  "index",
	"matrices": "matrix",
	"quizzes":  "quiz",
	"buses":    "bus",
}

func singular(word string) string {
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
