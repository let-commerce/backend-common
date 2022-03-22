package xstrings

import (
	"strings"
	"unicode"
)

// ToTitle Change string format to Title (e.g.: "test" -> "Test")
func ToTitle(str string) string {
	r := []rune(strings.ToLower(str))
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
