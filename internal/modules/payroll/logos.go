package payroll

import _ "embed"

//go:embed assets/sbm_logo.jpeg
var sbmLogo []byte

// schoolLogo returns the embedded logo bytes for a school, matched by name,
// or nil if that school has no logo on file.
func schoolLogo(schoolName string) []byte {
	if schoolName == "Shraddha Bal Mandir School" {
		return sbmLogo
	}
	return nil
}
