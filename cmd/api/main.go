package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/config"
	"github.com/GDGBabcockUniversity/auth-v2/internal/database"
	"github.com/GDGBabcockUniversity/auth-v2/internal/httpapi"
	"github.com/GDGBabcockUniversity/auth-v2/internal/repository"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	cfg, e := config.Load()

	if e != nil {
		panic(e)
	}

	ctx := context.Background()

	db, e := database.Connect(cfg.DatabaseURL)

	if e != nil {
		panic(e)
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if e = apply(ctx, db); e != nil {
			panic(e)
		}
		return
	}
	access := auth.NewManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudiences, cfg.AccessTTL)
	refresh := auth.NewManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudiences, cfg.RefreshTTL)
	api := httpapi.API{
		Auth: auth.Service{
			UserRepo:   repository.PostgresUserRepository{DB: db},
			Google:     auth.NewGoogleProvider(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL),
			Access:     access,
			Refresh:    refresh,
			RefreshTTL: cfg.RefreshTTL,
		},
		Audience:      cfg.JWTAudiences[0],
		Log:           slog.Default(),
		SecureCookies: cfg.Environment != "development",
	}
	if e = api.App().Listen(":" + cfg.Port); e != nil {
		panic(e)
	}
}
func apply(ctx context.Context, db *gorm.DB) error {
	entries, e := migrations.ReadDir("migrations")
	if e != nil {
		return e
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		b, e := migrations.ReadFile("migrations/" + entry.Name())
		if e != nil {
			return e
		}
		if e = db.WithContext(ctx).Exec(string(b)).Error; e != nil {
			return fmt.Errorf("%s: %w", entry.Name(), e)
		}
	}
	return nil
}
