package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/lib/pq"
)

// adminEditableProfileFields are the profile columns an admin may correct on
// someone else's record, alongside roles, teams and is_active.
var adminEditableProfileFields = map[string]bool{
	"full_name": true, "whatsapp_number": true, "avatar_url": true,
	"gender": true, "birthday": true, "student_status": true,
	"matric_no": true, "department": true, "faculty": true,
	"primary_track": true, "secondary_track": true,
	"primary_skill_level": true, "secondary_skill_level": true,
}

type Identity struct {
	UID, Email, Name, Picture string
	EmailVerified             bool
}
type Verifier interface {
	Verify(context.Context, string) (Identity, error)
}
type Service struct {
	Verifier   Verifier
	Repository domain.AuthRepository
	Tokens     Tokens
	Queue      domain.Queue
}
type Pair struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64
}

func NewAuthService(verifier Verifier, repository domain.AuthRepository, tokens Tokens, queue domain.Queue) *Service {
	return &Service{
		Verifier:   verifier,
		Repository: repository,
		Tokens:     tokens,
		Queue:      queue,
	}
}

func (s Service) Login(ctx context.Context, raw string, ip, agent *string) (domain.User, Pair, error) {
	identity, err := s.Verifier.Verify(ctx, raw)
	if err != nil {
		return domain.User{}, Pair{}, domain.ErrUnauthorized
	}
	if identity.UID == "" {
		return domain.User{}, Pair{}, domain.ErrUnauthorized
	}
	user, err := s.Repository.FindByFirebaseUID(ctx, identity.UID)
	if errors.Is(err, domain.ErrNotFound) {
		email := strings.ToLower(strings.TrimSpace(identity.Email))
		if email == "" || !strings.Contains(email, "@") {
			return domain.User{}, Pair{}, domain.ErrUnauthorized
		}
		var avatar *string
		if identity.Picture != "" {
			avatar = &identity.Picture
		}
		waiting, findErr := s.Repository.FindByEmail(ctx, email)
		switch {
		case findErr != nil && !errors.Is(findErr, domain.ErrNotFound):
			return domain.User{}, Pair{}, findErr

		case findErr == nil && !waiting.Pending():
			// Another Firebase UID already holds this email; needs linking.
			return domain.User{}, Pair{}, domain.ErrConflict

		case findErr == nil:
			// An admin pre-registered this person. Claim that row so the roles
			// and teams they were given survive their first sign-in.
			user, err = s.claimPending(ctx, waiting, identity, avatar)
			if err != nil {
				return domain.User{}, Pair{}, err
			}

		default:
			user = domain.User{
				FirebaseUID:   identity.UID,
				Email:         email,
				FullName:      identity.Name,
				AvatarURL:     avatar,
				EmailVerified: identity.EmailVerified,
				Roles:         []string{"user"},
				Teams:         []string{},
				IsActive:      true,
			}
			if user.FullName == "" {
				user.FullName = strings.Split(email, "@")[0]
			}
			created, createErr := s.Repository.Create(ctx, &user)
			if createErr != nil {
				return domain.User{}, Pair{}, createErr
			}
			if !created {
				user, err = s.Repository.FindByFirebaseUID(ctx, identity.UID)
				if errors.Is(err, domain.ErrNotFound) {
					return domain.User{}, Pair{}, domain.ErrConflict
				}
				if err != nil {
					return domain.User{}, Pair{}, err
				}
			}
		}
	} else if err != nil {
		return domain.User{}, Pair{}, err
	}
	if !user.IsActive {
		return domain.User{}, Pair{}, domain.ErrInactive
	}
	if err = s.Repository.UpdateLastLogin(ctx, user.ID); err != nil {
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
	if err = s.Repository.CreateRefreshToken(ctx, &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: Hash(refresh),
		ExpiresAt: time.Now().UTC().Add(s.Tokens.RefreshTTL),
		IsActive:  true,
		IPAddress: ip,
		UserAgent: agent,
	}); err != nil {
		return domain.User{}, Pair{}, err
	}
	if err := s.Repository.CreateAuditEvent(ctx, domain.AuditEvent{UserID: &user.ID, EventType: "login", IPAddress: ip, UserAgent: agent}); err != nil {
		slog.Error("login audit write failed", "error", err)
	}

	if s.Queue != nil {
		if err := s.Queue.Publish("user.upsert", map[string]any{
			"type": "user.upsert",
			"user": map[string]any{
				"id":              user.ID,
				"full_name":       user.FullName,
				"profile_picture": user.AvatarURL,
				"updated_at":      user.UpdatedAt,
			},
		}); err != nil {
			slog.Error("user event publish failed", "error", err)
		}
	}
	return user, Pair{access, refresh, "Bearer", int64(s.Tokens.AccessTTL.Seconds())}, nil
}
func (s Service) AccessUser(ctx context.Context, raw string) (domain.User, Claims, error) {
	c, err := s.Tokens.Verify(raw, "access")
	if err != nil {
		return domain.User{}, Claims{}, domain.ErrUnauthorized
	}
	u, err := s.Repository.FindByID(ctx, c.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, Claims{}, domain.ErrUnauthorized
	}
	if err != nil {
		return domain.User{}, Claims{}, err
	}
	if !u.IsActive {
		return domain.User{}, Claims{}, domain.ErrInactive
	}
	return u, c, nil
}
func (s Service) Refresh(ctx context.Context, raw string) (Pair, error) {
	c, e := s.Tokens.Verify(raw, "refresh")
	if e != nil {
		return Pair{}, domain.ErrUnauthorized
	}
	active, e := s.Repository.ExistsActiveRefreshToken(ctx, Hash(raw), c.UserID)
	if e != nil {
		return Pair{}, e
	}
	if !active {
		return Pair{}, domain.ErrUnauthorized
	}
	u, e := s.Repository.FindByID(ctx, c.UserID)
	if errors.Is(e, domain.ErrNotFound) {
		return Pair{}, domain.ErrUnauthorized
	}
	if e != nil {
		return Pair{}, e
	}
	if !u.IsActive {
		return Pair{}, domain.ErrUnauthorized
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
			return domain.ErrUnauthorized
		}
		hash = Hash(raw)
	}
	if e := s.Repository.RevokeRefreshTokens(ctx, userID, hash); e != nil {
		return e
	}
	if err := s.Repository.CreateAuditEvent(ctx, domain.AuditEvent{UserID: &userID, EventType: "logout"}); err != nil {
		slog.Error("logout audit write failed", "error", err)
	}
	return nil
}

func (s Service) UpdateProfile(ctx context.Context, id string, updates map[string]any) (domain.User, error) {
	allowed := map[string]bool{"full_name": true, "whatsapp_number": true, "avatar_url": true, "gender": true, "birthday": true, "student_status": true, "matric_no": true, "department": true, "faculty": true, "primary_track": true, "secondary_track": true, "primary_skill_level": true, "secondary_skill_level": true, "tos_agreed": true, "tos_agreed_at": true, "tos_version": true}
	if len(updates) == 0 {
		return domain.User{}, domain.ErrBadInput
	}
	for key, value := range updates {
		if !allowed[key] {
			return domain.User{}, domain.ErrBadInput
		}
		if key == "tos_agreed" {
			if _, ok := value.(bool); !ok {
				return domain.User{}, domain.ErrBadInput
			}
		} else if value != nil {
			if _, ok := value.(string); !ok {
				return domain.User{}, domain.ErrBadInput
			}
		}
	}
	return s.Repository.UpdateProfile(ctx, id, updates)
}

// claimPending hands an admin-created row its real Firebase identity, keeping
// the UUID and the roles and teams the admin already assigned to it.
func (s Service) claimPending(ctx context.Context, waiting domain.User, identity Identity, avatar *string) (domain.User, error) {
	updates := map[string]any{
		"firebase_uid":   identity.UID,
		"email_verified": identity.EmailVerified,
	}
	if identity.Name != "" {
		updates["full_name"] = identity.Name
	} else if waiting.FullName == "" {
		updates["full_name"] = strings.Split(waiting.Email, "@")[0]
	}
	if avatar != nil && waiting.AvatarURL == nil {
		updates["avatar_url"] = *avatar
	}
	return s.Repository.ClaimPendingUser(ctx, waiting.ID, updates)
}

// GetUser returns one user for the admin detail view, archived rows included.
func (s Service) GetUser(ctx context.Context, id string) (domain.User, error) {
	return s.Repository.FindByID(ctx, id)
}

// CreateUser pre-registers a member by email. The row waits under a sentinel
// firebase_uid until that person signs in with Google for the first time.
func (s Service) CreateUser(ctx context.Context, actor, email, fullName string, roles, teams []string) (domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return domain.User{}, domain.ErrBadInput
	}
	roles, err := cleanList(roles, []string{"user"})
	if err != nil {
		return domain.User{}, err
	}
	teams, err = cleanList(teams, []string{})
	if err != nil {
		return domain.User{}, err
	}
	exists, err := s.Repository.ExistsByEmail(ctx, email)
	if err != nil {
		return domain.User{}, err
	}
	if exists {
		return domain.User{}, domain.ErrConflict
	}

	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		fullName = strings.Split(email, "@")[0]
	}
	user := domain.User{
		FirebaseUID: domain.PendingUID(),
		Email:       email,
		FullName:    fullName,
		Roles:       roles,
		Teams:       teams,
		IsActive:    true,
	}
	created, err := s.Repository.Create(ctx, &user)
	if err != nil {
		return domain.User{}, err
	}
	if !created {
		return domain.User{}, domain.ErrConflict
	}
	data, err := json.Marshal(map[string]any{"target_user_id": user.ID, "email": user.Email, "roles": []string(user.Roles)})
	if err != nil {
		return domain.User{}, err
	}
	if err := s.Repository.CreateAuditEvent(ctx, domain.AuditEvent{UserID: &actor, EventType: "user_admin_create", EventData: data}); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// SetUserArchived soft-deletes or restores a user. Archiving revokes every
// refresh token so open sessions elsewhere stop working immediately.
func (s Service) SetUserArchived(ctx context.Context, actor, id string, archived bool) (domain.User, error) {
	if archived && actor == id {
		return domain.User{}, domain.ErrBadInput
	}
	user, err := s.Repository.SetUserDeleted(ctx, actor, id, archived)
	if err != nil {
		return domain.User{}, err
	}
	if archived {
		if err := s.Repository.RevokeRefreshTokens(ctx, id, ""); err != nil {
			return domain.User{}, err
		}
	}
	return user, nil
}

func cleanList(values []string, fallback []string) ([]string, error) {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, domain.ErrBadInput
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return fallback, nil
	}
	return out, nil
}

func (s Service) ListUsers(ctx context.Context, query domain.UserQuery) ([]domain.User, int64, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	return s.Repository.FindUsers(ctx, query)
}
func (s Service) UpdateUser(ctx context.Context, actor, id string, updates map[string]any) (domain.User, error) {
	if len(updates) == 0 {
		return domain.User{}, domain.ErrBadInput
	}
	for key, value := range updates {
		switch key {
		case "roles", "teams":
			values, ok := value.([]any)
			if !ok {
				return domain.User{}, domain.ErrBadInput
			}
			roles := make([]string, 0, len(values))
			for _, item := range values {
				role, ok := item.(string)
				if !ok || strings.TrimSpace(role) == "" {
					return domain.User{}, domain.ErrBadInput
				}
				roles = append(roles, role)
			}
			updates[key] = pq.StringArray(roles)
		case "is_active":
			if _, ok := value.(bool); !ok {
				return domain.User{}, domain.ErrBadInput
			}
		default:
			if !adminEditableProfileFields[key] {
				return domain.User{}, domain.ErrBadInput
			}
			text, ok := value.(string)
			if !ok {
				return domain.User{}, domain.ErrBadInput
			}
			updates[key] = strings.TrimSpace(text)
		}
	}
	return s.Repository.UpdateUser(ctx, actor, id, updates)
}
