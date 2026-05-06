// Package ui provides passphrase masking utilities.
package ui

import (
	"strings"
	"unicode/utf8"
)

// maskPassphrase replaces every rune in s with a bullet character ("•") for
// secure display in the passphrase input box.
// Per AUDIT.md HIGH finding: the previous inline loop was broken and produced
// at most a single bullet char; this pure function is also independently testable.
func maskPassphrase(s string) string {
	if s == "" {
		return s
	}
	return strings.Repeat("•", utf8.RuneCountInString(s))
}
