package auth

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/lib/pq"
)

var ErrUnauthorized = errors.New("unauthorized")
var ErrConflict = errors.New("identity conflict")
var ErrInactive = errors.New("inactive account")
var ErrBadInput = errors.New("invalid input")

type Identity struct {
	UID, Email, Name, Picture string
	EmailVerified             bool
}
type Verifier interface {
	Verify(context.Context, string) (Identity, error)
}
type Store interface {
	ByUID(context.Context, string) (domain.User, error)
	ByEmail(context.Context, string) (domain.User, error)
	ByID(context.Context, string) (domain.User, error)
	Create(context.Context, *domain.User) (bool, error)
	LastLogin(context.Context, string) error
	StoreRefresh(context.Context, *domain.RefreshToken) error
	ActiveRefresh(context.Context, string, string) (bool, error)
	RevokeRefresh(context.Context, string, string) error
	UpdateProfile(context.Context, string, map[string]any) (domain.User, error)
	AdminList(context.Context, string, string, int, int) ([]domain.User, int64, error)
	AdminUpdate(context.Context, string, string, map[string]any) (domain.User, error)
	Audit(context.Context, domain.AuditEvent) error
}
type Service struct {
	Verifier Verifier
	Store    Store
	Tokens   Tokens
}
type Pair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (s Service) Login(ctx context.Context, raw string, ip, agent *string) (domain.User, Pair, error) {
	identity, err := s.Verifier.Verify(ctx, raw)
	if err != nil {
		return domain.User{}, Pair{}, ErrUnauthorized
	}
	if identity.UID == "" {
		return domain.User{}, Pair{}, ErrUnauthorized
	}
	user, err := s.Store.ByUID(ctx, identity.UID)
	if errors.Is(err, domain.ErrNotFound) {
		email := strings.ToLower(strings.TrimSpace(identity.Email))
		if !identity.EmailVerified || email == "" || !strings.Contains(email, "@") {
			return domain.User{}, Pair{}, ErrUnauthorized
		}
		_, err = s.Store.ByEmail(ctx, email)
		if err == nil {
			return domain.User{}, Pair{}, ErrConflict
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, Pair{}, err
		}
		var avatar *string
		if identity.Picture != "" {
			avatar = &identity.Picture
		}
		user = domain.User{FirebaseUID: identity.UID, Email: email, FullName: identity.Name, AvatarURL: avatar, EmailVerified: true, Roles: []string{"user"}, Teams: []string{}, IsActive: true}
		if user.FullName == "" {
			user.FullName = strings.Split(email, "@")[0]
		}
		created, createErr := s.Store.Create(ctx, &user)
		if createErr != nil {
			return domain.User{}, Pair{}, createErr
		}
		if !created {
			user, err = s.Store.ByUID(ctx, identity.UID)
			if errors.Is(err, domain.ErrNotFound) {
				return domain.User{}, Pair{}, ErrConflict
			}
			if err != nil {
				return domain.User{}, Pair{}, err
			}
		}
	} else if err != nil {
		return domain.User{}, Pair{}, err
	}
	if !user.IsActive {
		return domain.User{}, Pair{}, ErrInactive
	}
	if err = s.Store.LastLogin(ctx, user.ID); err != nil {
		return domain.User{}, Pair{}, err
	}
	access, err := s.Tokens.Issue(user, "access")
	if err != nil {
		return domain.User{}, Pair{}, err
	}
	refresh, err := s.Tokens.Issue(user, "refresh")
	if err != nil {
		return domain.User{}, Pair{}, err
	}
	if err = s.Store.StoreRefresh(ctx, &domain.RefreshToken{UserID: user.ID, TokenHash: Hash(refresh), ExpiresAt: time.Now().UTC().Add(s.Tokens.RefreshTTL), IsActive: true, IPAddress: ip, UserAgent: agent}); err != nil {
		return domain.User{}, Pair{}, err
	}
	if err := s.Store.Audit(ctx, domain.AuditEvent{UserID: &user.ID, EventType: "login", IPAddress: ip, UserAgent: agent}); err != nil {
		slog.Error("login audit write failed", "error", err)
	}
	return user, Pair{access, refresh, "Bearer", int64(s.Tokens.AccessTTL.Seconds())}, nil
}
func (s Service) AccessUser(ctx context.Context, raw string) (domain.User, Claims, error) {
	c, err := s.Tokens.Verify(raw, "access")
	if err != nil {
		return domain.User{}, Claims{}, ErrUnauthorized
	}
	u, err := s.Store.ByID(ctx, c.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, Claims{}, ErrUnauthorized
	}
	if err != nil {
		return domain.User{}, Claims{}, err
	}
	if !u.IsActive {
		return domain.User{}, Claims{}, ErrInactive
	}
	return u, c, nil
}
func (s Service) Refresh(ctx context.Context, raw string) (Pair, error) {
	c, e := s.Tokens.Verify(raw, "refresh")
	if e != nil {
		return Pair{}, ErrUnauthorized
	}
	active, e := s.Store.ActiveRefresh(ctx, Hash(raw), c.UserID)
	if e != nil {
		return Pair{}, e
	}
	if !active {
		return Pair{}, ErrUnauthorized
	}
	u, e := s.Store.ByID(ctx, c.UserID)
	if errors.Is(e, domain.ErrNotFound) {
		return Pair{}, ErrUnauthorized
	}
	if e != nil {
		return Pair{}, e
	}
	if !u.IsActive {
		return Pair{}, ErrUnauthorized
	}
	access, e := s.Tokens.Issue(u, "access")
	if e != nil {
		return Pair{}, e
	}
	return Pair{access, raw, "Bearer", int64(s.Tokens.AccessTTL.Seconds())}, nil
}
func (s Service) Logout(ctx context.Context, userID, raw string) error {
	hash := ""
	if raw != "" {
		c, e := s.Tokens.Verify(raw, "refresh")
		if e != nil || c.UserID != userID {
			return ErrUnauthorized
		}
		hash = Hash(raw)
	}
	if e := s.Store.RevokeRefresh(ctx, userID, hash); e != nil {
		return e
	}
	if err := s.Store.Audit(ctx, domain.AuditEvent{UserID: &userID, EventType: "logout"}); err != nil {
		slog.Error("logout audit write failed", "error", err)
	}
	return nil
}

func (s Service) UpdateProfile(ctx context.Context, id string, updates map[string]any) (domain.User, error) {
	allowed := map[string]bool{"full_name": true, "whatsapp_number": true, "avatar_url": true, "gender": true, "birthday": true, "student_status": true, "matric_no": true, "department": true, "faculty": true, "primary_track": true, "secondary_track": true, "primary_skill_level": true, "secondary_skill_level": true, "tos_agreed": true, "tos_agreed_at": true, "tos_version": true}
	if len(updates) == 0 {
		return domain.User{}, ErrBadInput
	}
	for key, value := range updates {
		if !allowed[key] {
			return domain.User{}, ErrBadInput
		}
		if key == "tos_agreed" {
			if _, ok := value.(bool); !ok {
				return domain.User{}, ErrBadInput
			}
		} else if value != nil {
			if _, ok := value.(string); !ok {
				return domain.User{}, ErrBadInput
			}
		}
	}
	return s.Store.UpdateProfile(ctx, id, updates)
}
func (s Service) ListUsers(ctx context.Context, search, role string, page, limit int) ([]domain.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.Store.AdminList(ctx, search, role, page, limit)
}
func (s Service) UpdateUser(ctx context.Context, actor, id string, updates map[string]any) (domain.User, error) {
	if len(updates) == 0 {
		return domain.User{}, ErrBadInput
	}
	for key, value := range updates {
		switch key {
		case "roles", "teams":
			values, ok := value.([]any)
			if !ok {
				return domain.User{}, ErrBadInput
			}
			roles := make([]string, 0, len(values))
			for _, item := range values {
				role, ok := item.(string)
				if !ok || strings.TrimSpace(role) == "" {
					return domain.User{}, ErrBadInput
				}
				roles = append(roles, role)
			}
			updates[key] = pq.StringArray(roles)
		case "is_active":
			if _, ok := value.(bool); !ok {
				return domain.User{}, ErrBadInput
			}
		default:
			return domain.User{}, ErrBadInput
		}
	}
	return s.Store.AdminUpdate(ctx, actor, id, updates)
}
