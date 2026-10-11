package config_test

import (
	"strings"
	"testing"

	"github.com/GDGBabcockUniversity/auth-v2/internal/config"
)

func TestLoadRejectsExampleFirebaseConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("CORS_ORIGINS", "http://localhost:3000")
	t.Setenv("FIREBASE_PROJECT_ID", "your-firebase-project-id")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "FIREBASE_PROJECT_ID") {
		t.Fatalf("expected Firebase project configuration error, got %v", err)
	}
}
