package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/config"
	"github.com/GDGBabcockUniversity/auth-v2/internal/firebaseauth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/httpapi"
	"github.com/GDGBabcockUniversity/auth-v2/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("database initialization failed", "error", err)
		os.Exit(1)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		slog.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	verifier, err := firebaseauth.New(ctx, cfg)
	if err != nil {
		slog.Error("firebase initialization failed", "error", err)
		os.Exit(1)
	}
	service := auth.Service{Verifier: verifier, Store: repository.Store{DB: db}, Tokens: auth.Tokens{Secret: []byte(cfg.JWTSecret), AccessTTL: cfg.AccessTTL, RefreshTTL: cfg.RefreshTTL}}
	slog.Info("auth service listening", "port", cfg.Port)
	if err := (httpapi.API{Service: service, Origins: cfg.CORSOrigins}).App().Listen(":" + cfg.Port); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
