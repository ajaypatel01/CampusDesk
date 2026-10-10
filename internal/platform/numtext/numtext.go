// Package numtext tidies number-like text (phone, Aadhaar, Samagra, PEN...)
// that came through a spreadsheet: Excel stores long numbers as decimals,
// so "9876543210" arrives as "9876543210.0".
package numtext

import "strings"

// Clean trims spaces and drops a trailing ".0" (or ".00"...) when the rest
// is all digits. Anything else is returned trimmed but otherwise unchanged.
func Clean(s string) string {
	s = strings.TrimSpace(s)
	dot := strings.IndexByte(s, '.')
	if dot <= 0 || strings.Trim(s[dot+1:], "0") != "" || s[dot+1:] == "" {
		return s
	}
	for _, r := range s[:dot] {
		if r < '0' || r > '9' {
			return s
		}
	}
	return s[:dot]
}
