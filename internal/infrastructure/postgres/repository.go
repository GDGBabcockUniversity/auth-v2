package postgres

import (
	"context"
	"errors"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Repository struct {
	DB *gorm.DB
}

var _ domain.AuthRepository = Repository{}

func Open(ctx context.Context, dsn string) (Repository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return Repository{}, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return Repository{}, err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return Repository{}, err
	}
	return Repository{DB: db}, nil
}

func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}
	return err
}
