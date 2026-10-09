package domain

import (
	"context"
	"time"
)

type UserRepository interface {
	Create(context.Context, string, string, string) (User, error)
	FindOrCreateGoogleUser(context.Context, GoogleIdentity) (User, error)
	FindByEmail(context.Context, string) (User, error)
	FindByID(context.Context, string) (User, error)
	SaveRefresh(context.Context, string, string, time.Time) error
	ConsumeRefresh(context.Context, string) (User, error)

	List(ctx context.Context, limit, offset int, activeOnly *bool) ([]User, int64, error)
	SetActive(ctx context.Context, id string, active bool) (User, error)
	Delete(ctx context.Context, id string) error
}
