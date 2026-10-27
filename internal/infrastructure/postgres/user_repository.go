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
func (s Repository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	var u domain.User
	e := s.DB.WithContext(ctx).Where("lower(email) = ?", strings.ToLower(email)).First(&u).Error
	return u, notFound(e)
}

// ClaimPendingUser attaches a real Firebase identity to a row an admin
// pre-registered, keeping the UUID the admin already assigned roles to.
func (s Repository) ClaimPendingUser(ctx context.Context, id string, updates map[string]any) (domain.User, error) {
	if len(updates) == 0 {
		return domain.User{}, domain.ErrBadInput
	}
	if err := s.DB.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return domain.User{}, err
	}
	return s.FindByID(ctx, id)
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
func (s Repository) FindUsers(ctx context.Context, query domain.UserQuery) ([]domain.User, int64, error) {
	q := s.DB.WithContext(ctx).Model(&domain.User{})
	if query.Archived {
		q = q.Where("deleted_at IS NOT NULL")
	} else {
		q = q.Where("deleted_at IS NULL")
	}
	if query.Search != "" {
		q = q.Where("full_name ILIKE ? OR email ILIKE ?", "%"+query.Search+"%", "%"+query.Search+"%")
	}
	if query.Role != "" {
		q = q.Where("? = ANY(roles)", query.Role)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []domain.User
	e := q.Order("created_at DESC").Limit(query.Limit).Offset((query.Page - 1) * query.Limit).Find(&users).Error
	return users, total, e
}

// SetUserDeleted archives or restores a user. Archiving also deactivates the
// account, so the existing inactive check keeps their tokens from working.
func (s Repository) SetUserDeleted(ctx context.Context, actor, id string, deleted bool) (domain.User, error) {
	event := "user_admin_restore"
	updates := map[string]any{"deleted_at": nil, "is_active": true}
	if deleted {
		event = "user_admin_delete"
		updates = map[string]any{"deleted_at": time.Now().UTC(), "is_active": false}
	}
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before domain.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&before, "id = ?", id).Error; err != nil {
			return notFound(err)
		}
		if before.Deleted() == deleted {
			return domain.ErrBadInput
		}
		if err := tx.Model(&domain.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		data, err := json.Marshal(map[string]any{
			"target_user_id": id,
			"email":          before.Email,
			"roles":          before.Roles,
		})
		if err != nil {
			return err
		}
		return tx.Create(&domain.AuditEvent{UserID: &actor, EventType: event, EventData: data}).Error
	})
	if err != nil {
		return domain.User{}, err
	}
	return s.FindByID(ctx, id)
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
