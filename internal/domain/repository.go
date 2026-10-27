package domain

import "context"

// UserQuery filters the admin roster. Archived rows are excluded unless
// Archived is set, so a soft delete removes a member from the default view.
type UserQuery struct {
	Search, Role string
	Archived     bool
	Page, Limit  int
}

type UserRepository interface {
	FindByID(context.Context, string) (User, error)
	FindByFirebaseUID(context.Context, string) (User, error)
	FindByEmail(context.Context, string) (User, error)
	ExistsByEmail(context.Context, string) (bool, error)
	Create(context.Context, *User) (bool, error)
	ClaimPendingUser(context.Context, string, map[string]any) (User, error)
	UpdateLastLogin(context.Context, string) error
	UpdateProfile(context.Context, string, map[string]any) (User, error)
	FindUsers(context.Context, UserQuery) ([]User, int64, error)
	UpdateUser(context.Context, string, string, map[string]any) (User, error)
	SetUserDeleted(context.Context, string, string, bool) (User, error)
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
