package domain

import "context"

// UserRepository is the persistence contract used by the auth domain.
type UserRepository interface {
	FindByID(context.Context, string) (User, error)
	FindByFirebaseUID(context.Context, string) (User, error)
	ExistsByEmail(context.Context, string) (bool, error)
	Create(context.Context, *User) (bool, error)
	UpdateLastLogin(context.Context, string) error
	UpdateProfile(context.Context, string, map[string]any) (User, error)
	FindUsers(context.Context, string, string, int, int) ([]User, int64, error)
	UpdateUser(context.Context, string, string, map[string]any) (User, error)
}

type RefreshTokenRepository interface {
	CreateRefreshToken(context.Context, *RefreshToken) error
	ExistsActiveRefreshToken(context.Context, string, string) (bool, error)
	RevokeRefreshTokens(context.Context, string, string) error
}

type AuditRepository interface {
	CreateAuditEvent(context.Context, AuditEvent) error
}

type AuthRepository interface {
	UserRepository
	RefreshTokenRepository
	AuditRepository
}
