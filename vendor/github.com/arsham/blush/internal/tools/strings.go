package tools

import (
	"unicode"
)

// IsPlainText returns false if at least one of the runes in the input is not
// represented as a plain text in a file. Null is an exception.
func IsPlainText(input string) bool {
	for _, r := range input {
		switch r {
		case 0, '\n', '\t', '\r':
			continue
		}
		if r > unicode.MaxASCII || !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
