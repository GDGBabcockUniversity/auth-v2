// Package database owns application database connection configuration.
package database

import (
	"fmt"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL database: %w", err)
	}

	db.AutoMigrate(domain.User{})
	return db, nil
}
