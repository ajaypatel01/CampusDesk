package numtext

import "testing"

func TestClean(t *testing.T) {
	for in, want := range map[string]string{
		"9876543210.0": "9876543210", " 123456789.00 ": "123456789", "9876543210": "9876543210",
		"12.5": "12.5", "A12.0": "A12.0", "1.": "1.", ".0": ".0", "": "", "Ram.0": "Ram.0", "98-76.0": "98-76.0",
	} {
		if got := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}
