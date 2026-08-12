package view

import "regexp"

var compiled = map[string]*regexp.Regexp{}

func mustCompile(pattern string) *regexp.Regexp {
	if re, ok := compiled[pattern]; ok {
		return re
	}
	re := regexp.MustCompile(pattern)
	compiled[pattern] = re
	return re
}
