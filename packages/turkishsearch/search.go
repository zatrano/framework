package turkishsearch

import (
	"strings"
	"unicode"
)

func normalize(str string) string {
	replacements := map[rune]rune{
		'ç': 'c', 'Ç': 'C',
		'ğ': 'g', 'Ğ': 'G',
		'ı': 'i', 'I': 'I',
		'ö': 'o', 'Ö': 'O',
		'ş': 's', 'Ş': 'S',
		'ü': 'u', 'Ü': 'U',
	}

	var builder strings.Builder
	for _, r := range str {
		if repl, ok := replacements[r]; ok {
			builder.WriteRune(unicode.ToLower(repl))
		} else {
			builder.WriteRune(unicode.ToLower(r))
		}
	}
	return builder.String()
}

func MatchNormalized(text, keyword string) bool {
	normText := normalize(text)
	normKeyword := normalize(keyword)
	return strings.Contains(normText, normKeyword)
}

func SQLFilter(columnName, search string) (string, []interface{}) {
	filterValue := "%" + strings.ToLower(search) + "%"

	query := "unaccent(lower(" + columnName + ")) ILIKE unaccent($1)"

	params := []interface{}{filterValue}

	return query, params
}
