package payroll

import _ "embed"

//go:embed assets/sbm_logo.jpeg
var sbmLogo []byte

//go:embed assets/fws_logo.jpeg
var fwsLogo []byte

// schoolLogo returns the embedded logo bytes for a school, matched by name,
// or nil if that school has no logo on file.
func schoolLogo(schoolName string) []byte {
	switch schoolName {
	case "Shraddha Bal Mandir School":
		return sbmLogo
	case "Freedom World School Bhatkhedi":
		return fwsLogo
	}
	return nil
}
