package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct{ DB *gorm.DB }

func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}
	return err
}

func (s Store) ByUID(ctx context.Context, uid string) (domain.User, error) {
	var u domain.User
	e := s.DB.WithContext(ctx).Where("firebase_uid = ?", uid).First(&u).Error
	return u, notFound(e)
}
func (s Store) ByEmail(ctx context.Context, email string) (domain.User, error) {
	var u domain.User
	e := s.DB.WithContext(ctx).Where("lower(email) = ?", strings.ToLower(email)).First(&u).Error
	return u, notFound(e)
}
func (s Store) ByID(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	e := s.DB.WithContext(ctx).First(&u, "id = ?", id).Error
	return u, notFound(e)
}
func (s Store) Create(ctx context.Context, u *domain.User) (bool, error) {
	r := s.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(u)
	return r.RowsAffected == 1, r.Error
}
func (s Store) LastLogin(ctx context.Context, id string) error {
	return s.DB.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Update("last_login_at", time.Now().UTC()).Error
}
func (s Store) StoreRefresh(ctx context.Context, t *domain.RefreshToken) error {
	return s.DB.WithContext(ctx).Create(t).Error
}
func (s Store) ActiveRefresh(ctx context.Context, hash string, userID string) (bool, error) {
	var count int64
	e := s.DB.WithContext(ctx).Model(&domain.RefreshToken{}).Where("token_hash = ? AND user_id = ? AND is_active = true AND expires_at > ?", hash, userID, time.Now().UTC()).Count(&count).Error
	return count > 0, e
}
func (s Store) RevokeRefresh(ctx context.Context, userID, hash string) error {
	q := s.DB.WithContext(ctx).Model(&domain.RefreshToken{}).Where("user_id = ? AND is_active = true", userID)
	if hash != "" {
		q = q.Where("token_hash = ?", hash)
	}
	return q.Updates(map[string]any{"is_active": false, "revoked_at": time.Now().UTC()}).Error
}
func (s Store) UpdateProfile(ctx context.Context, id string, updates map[string]any) (domain.User, error) {
	if len(updates) == 0 {
		return domain.User{}, errors.New("no profile fields")
	}
	if err := s.DB.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return domain.User{}, err
	}
	return s.ByID(ctx, id)
}
func (s Store) AdminList(ctx context.Context, search, role string, page, limit int) ([]domain.User, int64, error) {
	q := s.DB.WithContext(ctx).Model(&domain.User{})
	if search != "" {
		q = q.Where("full_name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if role != "" {
		q = q.Where("? = ANY(roles)", role)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []domain.User
	e := q.Order("created_at DESC").Limit(limit).Offset((page - 1) * limit).Find(&users).Error
	return users, total, e
}
func (s Store) AdminUpdate(ctx context.Context, actor, id string, updates map[string]any) (domain.User, error) {
	if len(updates) == 0 {
		return domain.User{}, errors.New("no admin fields")
	}
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before domain.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&before, "id = ?", id).Error; err != nil {
			return notFound(err)
		}
		q := tx.Model(&domain.User{}).Where("id = ?", id).Updates(updates)
		if q.Error != nil {
			return q.Error
		}
		var after domain.User
		if err := tx.First(&after, "id = ?", id).Error; err != nil {
			return notFound(err)
		}
		data, err := json.Marshal(map[string]any{
			"target_user_id": id,
			"before":         map[string]any{"roles": before.Roles, "teams": before.Teams, "is_active": before.IsActive},
			"after":          map[string]any{"roles": after.Roles, "teams": after.Teams, "is_active": after.IsActive},
		})
		if err != nil {
			return err
		}
		return tx.Create(&domain.AuditEvent{UserID: &actor, EventType: "user_admin_update", EventData: data}).Error
	})
	if err != nil {
		return domain.User{}, err
	}
	return s.ByID(ctx, id)
}
func (s Store) Audit(ctx context.Context, event domain.AuditEvent) error {
	return s.DB.WithContext(ctx).Create(&event).Error
}
