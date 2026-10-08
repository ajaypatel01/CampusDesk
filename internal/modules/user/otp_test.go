package user

import (
	"reflect"
	"testing"
)

func TestMobileNumbers(t *testing.T) {
	cases := map[string][]string{
		"9876543210":                          {"9876543210"},
		"+91 98765 43210":                     {"9876543210"},
		"+91-9876543210":                      {"9876543210"},
		"09876543210":                         {"9876543210"},
		"919876543210":                        {"9876543210"},
		"9876543210 / 9123456789":             {"9876543210", "9123456789"},
		"9876543210, 09123456789":             {"9876543210", "9123456789"},
		"9876543210 9123456789":               {"9876543210", "9123456789"},
		"Father 9876543210 Mother 9123456789": {"9876543210", "9123456789"},
		"0 9876543210":                        {"9876543210"},
		"098765 43210":                        {"9876543210"},
		"0755-2345678":                        nil, // landline
		"(0755) 2345678":                      nil,
		"12345":                               nil,
		"":                                    nil,
	}
	for in, want := range cases {
		if got := mobileNumbers(in); !reflect.DeepEqual(got, want) {
			t.Errorf("mobileNumbers(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNormalizePhone(t *testing.T) {
	for _, in := range []string{"9876543210", "+91 98765 43210", "098765-43210"} {
		if got, err := normalizePhone(in); err != nil || got != "9876543210" {
			t.Errorf("normalizePhone(%q) = %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"", "12345", "1234567890", "9876543210 9123456789"} {
		if _, err := normalizePhone(in); err == nil {
			t.Errorf("normalizePhone(%q) should fail", in)
		}
	}
}
