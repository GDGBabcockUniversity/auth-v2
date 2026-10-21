package config_test

import (
	"strings"
	"testing"

	"github.com/GDGBabcockUniversity/auth-v2/internal/config"
)

func TestLoadPort(t *testing.T) {
	for _, tc := range []struct {
		name, port, httpPort, want string
	}{
		{"Vercel port takes precedence", "3000", "8080", "3000"},
		{"local HTTP port", "", "9090", "9090"},
		{"default port", "", "", "8080"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PORT", tc.port)
			t.Setenv("HTTP_PORT", tc.httpPort)
			t.Setenv("DATABASE_URL", "postgres://example")
			t.Setenv("CORS_ORIGINS", "http://localhost:3000")
			t.Setenv("FIREBASE_PROJECT_ID", "real-project-id")
			t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
			t.Setenv("JWT_ACCESS_TTL", "15m")
			t.Setenv("JWT_REFRESH_TTL", "168h")
			cfg, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Port != tc.want {
				t.Fatalf("port = %q, want %q", cfg.Port, tc.want)
			}
		})
	}
}

func TestLoadRejectsExampleFirebaseConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("CORS_ORIGINS", "http://localhost:3000")
	t.Setenv("FIREBASE_PROJECT_ID", "your-firebase-project-id")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "FIREBASE_PROJECT_ID") {
		t.Fatalf("expected Firebase project configuration error, got %v", err)
	}
}

func TestLoadRejectsExampleJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("CORS_ORIGINS", "http://localhost:3000")
	t.Setenv("FIREBASE_PROJECT_ID", "real-project-id")
	t.Setenv("JWT_SECRET", "replace-with-a-long-random-secret-at-least-32-characters")
	if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT secret configuration error, got %v", err)
	}
}
