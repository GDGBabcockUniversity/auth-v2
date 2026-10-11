package postgres

import (
	"context"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
)

func (s Repository) CreateRefreshToken(ctx context.Context, t *domain.RefreshToken) error {
	return s.DB.WithContext(ctx).Create(t).Error
}
func (s Repository) ExistsActiveRefreshToken(ctx context.Context, hash string, userID string) (bool, error) {
	var count int64
	e := s.DB.WithContext(ctx).Model(&domain.RefreshToken{}).Where("token_hash = ? AND user_id = ? AND is_active = true AND expires_at > ?", hash, userID, time.Now().UTC()).Count(&count).Error
	return count > 0, e
}
func (s Repository) RevokeRefreshTokens(ctx context.Context, userID, hash string) error {
	q := s.DB.WithContext(ctx).Model(&domain.RefreshToken{}).Where("user_id = ? AND is_active = true", userID)
	if hash != "" {
		q = q.Where("token_hash = ?", hash)
	}
	return q.Updates(map[string]any{"is_active": false, "revoked_at": time.Now().UTC()}).Error
}
