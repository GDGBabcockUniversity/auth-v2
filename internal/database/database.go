package database

import (
	"context"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Open(ctx context.Context, dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(domain.AuditEvent{}, domain.RefreshToken{}, domain.User{}, domain.Session{})

	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()

	if err != nil {
		return nil, err
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}
