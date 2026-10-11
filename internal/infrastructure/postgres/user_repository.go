package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s Repository) FindByFirebaseUID(ctx context.Context, uid string) (domain.User, error) {
	var u domain.User
	e := s.DB.WithContext(ctx).Where("firebase_uid = ?", uid).First(&u).Error
	return u, notFound(e)
}
func (s Repository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := s.DB.WithContext(ctx).Model(&domain.User{}).Where("lower(email) = ?", strings.ToLower(email)).Count(&count).Error
	return count > 0, err
}
func (s Repository) FindByID(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	e := s.DB.WithContext(ctx).First(&u, "id = ?", id).Error
	return u, notFound(e)
}
func (s Repository) Create(ctx context.Context, u *domain.User) (bool, error) {
	r := s.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(u)
	return r.RowsAffected == 1, r.Error
}
func (s Repository) UpdateLastLogin(ctx context.Context, id string) error {
	return s.DB.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Update("last_login_at", time.Now().UTC()).Error
}
func (s Repository) UpdateProfile(ctx context.Context, id string, updates map[string]any) (domain.User, error) {
	if len(updates) == 0 {
		return domain.User{}, domain.ErrBadInput
	}
	if err := s.DB.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return domain.User{}, err
	}
	return s.FindByID(ctx, id)
}
func (s Repository) FindUsers(ctx context.Context, search, role string, page, limit int) ([]domain.User, int64, error) {
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
func (s Repository) UpdateUser(ctx context.Context, actor, id string, updates map[string]any) (domain.User, error) {
	if len(updates) == 0 {
		return domain.User{}, domain.ErrBadInput
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
	return s.FindByID(ctx, id)
}
