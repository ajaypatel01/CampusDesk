package user

import "testing"

func TestIsDefaultPassword(t *testing.T) {
	kids := []string{"Ajay", "riya", "Ajay Kumar ", "VIHAAN"}
	for _, p := range []string{"Ajay@123", "ajay@123", "AJAY@123", "Riya@123", "AjayKumar@123", "Vihaan@123"} {
		if !isDefaultPassword(p, kids) {
			t.Errorf("%q should match", p)
		}
	}
	for _, p := range []string{"Ajay@1234", "Ajay123", "@123", "Rahul@123", "Ajay Kumar@123x", "Kumar@123", ""} {
		if isDefaultPassword(p, kids) {
			t.Errorf("%q should not match", p)
		}
	}
	if isDefaultPassword("Ajay@123", nil) {
		t.Error("no children: nothing matches")
	}
}
