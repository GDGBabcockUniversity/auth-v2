package postgres

import (
	"context"
	"errors"

	"github.com/GDGBabcockUniversity/auth-v2/internal/database"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"gorm.io/gorm"
)

type Repository struct {
	DB *gorm.DB
}

var _ domain.AuthRepository = Repository{}

func Open(ctx context.Context, dsn string) (Repository, error) {
	db, err := database.Open(ctx, dsn)
	if err != nil {
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
