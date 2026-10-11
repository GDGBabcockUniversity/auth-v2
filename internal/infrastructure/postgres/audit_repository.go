package postgres

import (
	"context"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
)

func (s Repository) CreateAuditEvent(ctx context.Context, event domain.AuditEvent) error {
	return s.DB.WithContext(ctx).Create(&event).Error
}
