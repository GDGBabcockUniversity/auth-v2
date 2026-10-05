package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
)

func TestJWTRequiresExpectedURIAndTokenType(t *testing.T) {
	m := NewManager("12345678901234567890123456789012", "https://auth.gdgbabcock.com", []string{"https://gdgbabcock.com", "https://radar.gdgbabcock.com"}, time.Hour)
	token, err := m.Issue("u1", []string{"user"}, "access")
	if err != nil {
		t.Fatal(err)
	}
	c, err := m.Verify(token, "https://radar.gdgbabcock.com", "access")
	if err != nil || c.Subject != "u1" {
		t.Fatalf("valid token: %v", err)
	}
	_, err = m.Verify(token, "https://orbit.gdgbabcock.com", "access")
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("wrong audience: %v", err)
	}
	_, err = m.Verify(token, "https://gdgbabcock.com", "refresh")
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("wrong type: %v", err)
	}
}
