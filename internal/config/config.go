package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Environment        string
	Port               string
	DatabaseURL        string
	JWTIssuer          string
	JWTSecret          string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	CORSOrigins        []string
	JWTAudiences       []string
	AccessTTL          time.Duration
	RefreshTTL         time.Duration
}

func Load() (Config, error) {
	c := Config{
		Environment:        getEnv("APP_ENV", "development"),
		Port:               getEnv("HTTP_PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		JWTIssuer:          getEnv("JWT_ISSUER", "https://auth.gdgbabcock.com"),
		JWTSecret:          getEnv("JWT_HMAC_SECRET", ""),
		CORSOrigins:        list("CORS_ORIGINS"),
		JWTAudiences:       list("JWT_AUDIENCES"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
	}

	var err error
	if c.AccessTTL, err = time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m")); err != nil {
		return c, fmt.Errorf("JWT_ACCESS_TTL: %w", err)
	}
	if c.RefreshTTL, err = time.ParseDuration(getEnv("JWT_REFRESH_TTL", "168h")); err != nil {
		return c, fmt.Errorf("JWT_REFRESH_TTL: %w", err)
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.JWTSecret) < 32 {
		return c, fmt.Errorf("JWT_HMAC_SECRET must be at least 32 characters")
	}
	if len(c.JWTAudiences) == 0 {
		return c, fmt.Errorf("JWT_AUDIENCES is required")
	}
	googleValues := []string{c.GoogleClientID, c.GoogleClientSecret, c.GoogleRedirectURL}
	configuredGoogleValues := 0
	for _, value := range googleValues {
		if value != "" {
			configuredGoogleValues++
		}
	}
	if configuredGoogleValues != 0 && configuredGoogleValues != len(googleValues) {
		return c, fmt.Errorf("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URL must be set together")
	}
	return c, nil
}

func getEnv(value, defaultValue string) string {
	if v := os.Getenv(value); v != "" {
		return v
	}
	return defaultValue
}
func list(k string) []string {
	var out []string
	for v := range strings.SplitSeq(os.Getenv(k), ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
