package auth

import "testing"

func TestPasswordHashRoundTrip(t *testing.T) {
	h, e := HashPassword("a secure password")
	if e != nil {
		t.Fatal(e)
	}
	if !VerifyPassword(h, "a secure password") {
		t.Fatal("expected matching password")
	}
	if VerifyPassword(h, "wrong password") {
		t.Fatal("wrong password accepted")
	}
}
