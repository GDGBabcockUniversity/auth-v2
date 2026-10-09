package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresUserRepository struct {
	DB *gorm.DB
}

func (r PostgresUserRepository) Create(ctx context.Context, email, name, passwordHash string) (domain.User, error) {
	u := domain.User{Email: email, FullName: name, PasswordHash: passwordHash, Roles: []string{"USER"}, Active: true}
	err := r.DB.WithContext(ctx).Create(&u).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return u, domain.ErrConflict
	}
	return u, err
}

func (r PostgresUserRepository) FindOrCreateGoogleUser(ctx context.Context, identity domain.GoogleIdentity) (domain.User, error) {
	identity.Email = strings.TrimSpace(strings.ToLower(identity.Email))
	identity.Name = strings.TrimSpace(identity.Name)
	var user domain.User

	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("google_subject = ?", identity.Subject).First(&user).Error; err == nil {
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// A verified Google email may safely link a pre-existing password account.
		err := tx.Where("email = lower(?)", identity.Email).First(&user).Error
		if err == nil {
			updates := map[string]any{"google_subject": identity.Subject, "email_verified": true, "profile_picture": identity.Picture}
			if user.FullName == "" && identity.Name != "" {
				updates["full_name"] = identity.Name
			}
			return tx.Model(&user).Updates(updates).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		user = domain.User{
			Email:          identity.Email,
			FullName:       identity.Name,
			GoogleSubject:  &identity.Subject,
			Roles:          []string{"USER"},
			Active:         true,
			EmailVerified:  true,
			ProfilePicture: &identity.Picture,
		}

		return tx.Create(&user).Error
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		// A concurrent callback linked this Google identity; return that account.
		return r.FindByGoogleSubject(ctx, identity.Subject)
	}
	return user, err
}

func (r PostgresUserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.by(ctx, "email = lower(?)", email)
}

func (r PostgresUserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	return r.by(ctx, "id = ?", id)
}

func (r PostgresUserRepository) FindByGoogleSubject(ctx context.Context, subject string) (domain.User, error) {
	return r.by(ctx, "google_subject = ?", subject)
}

func (r PostgresUserRepository) by(ctx context.Context, q string, arg any) (domain.User, error) {
	u := domain.User{}
	err := r.DB.WithContext(ctx).Where(q, arg).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return u, domain.ErrNotFound
	}
	return u, err
}

func (r PostgresUserRepository) SaveRefresh(ctx context.Context, userID, hash string, expires time.Time) error {
	return r.DB.WithContext(ctx).Create(&domain.RefreshToken{UserID: userID, TokenHash: hash, ExpiresAt: expires}).Error
}

func (r PostgresUserRepository) ConsumeRefresh(ctx context.Context, hash string) (domain.User, error) {
	var user domain.User
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

func (r PostgresUserRepository) List(ctx context.Context, limit, offset int, activeOnly *bool) ([]domain.User, int64, error){

	if limit <= 0 || limit > 100{
		limit = 20
	}

	if offset < 0{
		offset = 0
	}

	q := r.DB.WithContext(ctx).Model(&domain.User{})
	if activeOnly != nil{
		q = q.Where("is_active = ?", *activeOnly)
	}

	var users []domain.User
	var total int64

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := q.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error
	if err != nil{
		return nil, 0, err
	}
	
	return users, total, nil
}

func (r PostgresUserRepository) SetActive(ctx context.Context, id string, active bool) (domain.User, error) {
	var user domain.User
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error{
		res := tx.Model(&domain.User{}).
				Where("id = ?", id).
				Updates(map[string]any{
					"is_active": active,
					"updated_at": time.Now().UTC(),
				})

				if res.Error != nil{
					return res.Error
				}
				if res.RowsAffected == 0{
					return domain.ErrNotFound
				}

				if !active {
					now := time.Now().UTC()
					if err := tx.Model(&domain.RefreshToken{}).
								Where("user_id = ? AND revoked_at IS NULL", id).
								Update("revoked_at", now).Error; err != nil{
									return err
								}
				}

				return tx.Where("id = ?", id).First(&user).Error
	})
	return user, err
}
func (r PostgresUserRepository) Delete(ctx context.Context, id string) error {
	res := r.DB.WithContext(ctx).Delete(&domain.User{}, "id = ?", id)
	if res.Error != nil{
		return res.Error
	}
	if res.RowsAffected == 0{
		return domain.ErrNotFound
	}

	return nil
} 