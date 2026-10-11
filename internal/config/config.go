package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port, DatabaseURL, FirebaseProjectID, FirebaseServiceAccount, FirebaseServiceAccountPath, JWTSecret string
	CORSOrigins                                                                                         []string
	AccessTTL, RefreshTTL                                                                               time.Duration
}

func Load() (Config, error) {
	c := Config{Port: env("HTTP_PORT", "8080"), DatabaseURL: os.Getenv("DATABASE_URL"), FirebaseProjectID: os.Getenv("FIREBASE_PROJECT_ID"), FirebaseServiceAccount: os.Getenv("FIREBASE_SERVICE_ACCOUNT"), FirebaseServiceAccountPath: os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH"), JWTSecret: os.Getenv("JWT_SECRET"), CORSOrigins: split(os.Getenv("CORS_ORIGINS"))}
	if c.DatabaseURL == "" || c.FirebaseProjectID == "" || len(c.JWTSecret) < 32 {
		return c, errors.New("DATABASE_URL, FIREBASE_PROJECT_ID, and JWT_SECRET (32+ characters) are required")
	}
	var err error
	if c.AccessTTL, err = time.ParseDuration(env("JWT_ACCESS_TTL", "24h")); err != nil || c.AccessTTL <= 0 {
		return c, errors.New("invalid JWT_ACCESS_TTL")
	}
	if c.RefreshTTL, err = time.ParseDuration(env("JWT_REFRESH_TTL", "168h")); err != nil || c.RefreshTTL <= 0 {
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
func env(key, def string) string {
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
