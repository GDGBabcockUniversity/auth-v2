package store

import (
	"context"
	"errors"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserStore struct {
	DB *gorm.DB
}

func (s UserStore) Create(ctx context.Context, email, name, passwordHash string) (domain.User, error) {
	u := domain.User{Email: email, FullName: name, PasswordHash: passwordHash, Roles: []string{"USER"}, Active: true}
	err := s.DB.WithContext(ctx).Create(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return u, domain.ErrConflict
		}
		return u, err
	}
	return u, nil
}
func (s UserStore) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	return s.by(ctx, "email = lower(?)", email)
}
func (s UserStore) FindByID(ctx context.Context, id string) (domain.User, error) {
	return s.by(ctx, "id = ?", id)
}
func (s UserStore) by(ctx context.Context, q string, arg any) (domain.User, error) {
	u := domain.User{}
	err := s.DB.WithContext(ctx).Where(q, arg).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return u, domain.ErrNotFound
	}
	return u, err
}
func (s UserStore) SaveRefresh(ctx context.Context, userID, hash string, expires time.Time) error {
	return s.DB.WithContext(ctx).Create(&domain.RefreshToken{UserID: userID, TokenHash: hash, ExpiresAt: expires}).Error
}
func (s UserStore) ConsumeRefresh(ctx context.Context, hash string) (domain.User, error) {
	var user domain.User
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var token domain.RefreshToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ? AND revoked_at IS NULL AND expires_at > now()", hash).
			First(&token).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrRefreshReuse
			}
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&token).Update("revoked_at", now).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", token.UserID).First(&user).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, domain.ErrNotFound
	}
	return user, err
}
