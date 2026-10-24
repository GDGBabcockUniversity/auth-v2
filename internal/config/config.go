package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port, DatabaseURL, FirebaseProjectID, FirebaseServiceAccount, JWTSecret, QSTASH_TOKEN string
	CORSOrigins                                                                                                       []string
	AccessTTL, RefreshTTL                                                                                             time.Duration
}

func Load() (Config, error) {
	c := Config{
		Port:                       getEnv("PORT", getEnv("HTTP_PORT", "8080")),
		DatabaseURL:                getEnv("DATABASE_URL", ""),
		FirebaseProjectID:          getEnv("FIREBASE_PROJECT_ID", ""),
		FirebaseServiceAccount:     getEnv("FIREBASE_SERVICE_ACCOUNT", ""),
		JWTSecret:                  getEnv("JWT_SECRET", ""),
		CORSOrigins:                split(getEnv("CORS_ORIGINS", "")),
		QSTASH_TOKEN:               getEnv("QSTASH_TOKEN", ""),
	}

	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL is required")
	}
	if c.FirebaseProjectID == "" || c.FirebaseProjectID == "your-firebase-project-id" {
		return c, errors.New("FIREBASE_PROJECT_ID must be set to your Firebase project ID")
	}
	if len(c.JWTSecret) < 32 || strings.HasPrefix(c.JWTSecret, "replace-with-") {
		return c, errors.New("JWT_SECRET must be a non-placeholder secret of at least 32 characters")
	}
	var err error
	if c.AccessTTL, err = time.ParseDuration(getEnv("JWT_ACCESS_TTL", "24h")); err != nil || c.AccessTTL <= 0 {
		return c, errors.New("invalid JWT_ACCESS_TTL")
	}
	if c.RefreshTTL, err = time.ParseDuration(getEnv("JWT_REFRESH_TTL", "168h")); err != nil || c.RefreshTTL <= 0 {
		return c, errors.New("invalid JWT_REFRESH_TTL")
	}
	if len(c.CORSOrigins) == 0 {
		return c, errors.New("CORS_ORIGINS is required")
	}
	for _, origin := range c.CORSOrigins {
		if origin == "*" {
			return c, errors.New("CORS_ORIGINS must list explicit origins")
		}
	}
	return c, nil
}
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func split(s string) []string {
	var out []string
	for v := range strings.SplitSeq(s, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
