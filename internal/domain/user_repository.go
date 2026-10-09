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
}

type GoogleHandoffRepository interface {
	CreateGoogleHandoff(context.Context, string, string, time.Time) error
	ConsumeGoogleHandoff(context.Context, string) (User, error)
}
