package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/config"
	"github.com/GDGBabcockUniversity/auth-v2/internal/httpapi"
	"github.com/GDGBabcockUniversity/auth-v2/internal/infrastructure/firebase"
	"github.com/GDGBabcockUniversity/auth-v2/internal/infrastructure/postgres"
	"github.com/GDGBabcockUniversity/auth-v2/internal/infrastructure/queue"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	repository, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	verifier, err := firebase.New(ctx, cfg)
	if err != nil {
		slog.Error("firebase initialization failed", "error", err)
		os.Exit(1)
	}

	publisher := queue.NewClient(cfg.QSTASH_TOKEN)

	service := auth.NewAuthService(verifier, repository, auth.Tokens{
		Secret:     []byte(cfg.JWTSecret),
		AccessTTL:  cfg.AccessTTL,
		RefreshTTL: cfg.RefreshTTL,
	}, publisher)

	slog.Info("auth service listening", "port", cfg.Port)
	if err := (httpapi.API{Service: *service, Origins: cfg.CORSOrigins}).App().Listen(":" + cfg.Port); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
