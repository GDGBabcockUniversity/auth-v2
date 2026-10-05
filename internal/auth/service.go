package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/google/uuid"
)

type Service struct {
	UserRepo        domain.UserRepository
	Access, Refresh *Manager
	RefreshTTL      time.Duration
}

type Session struct {
	User         domain.User `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int64       `json:"expires_in"`
}

func (s Service) Register(ctx context.Context, email, name, password string) (Session, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if !strings.Contains(email, "@") {
		return Session{}, domain.FieldError{Field: "email", Message: "must be valid"}
	}
	hash, e := HashPassword(password)
	if e != nil {
		return Session{}, domain.FieldError{Field: "password", Message: e.Error()}
	}
	u, e := s.UserRepo.Create(ctx, email, strings.TrimSpace(name), hash)
	if e != nil {
		return Session{}, e
	}
	return s.issue(ctx, u)
}
func (s Service) Login(ctx context.Context, email, password string) (Session, error) {
	u, e := s.UserRepo.FindByEmail(ctx, email)
	if e != nil || !VerifyPassword(u.PasswordHash, password) {
		return Session{}, domain.ErrUnauthorized
	}
	if !u.Active {
		return Session{}, domain.ErrInactiveAccount
	}
	return s.issue(ctx, u)
}
func (s Service) RefreshSession(ctx context.Context, raw string) (Session, error) {
	u, e := s.UserRepo.ConsumeRefresh(ctx, hash(raw))
	if e != nil {
		return Session{}, e
	}
	if !u.Active {
		return Session{}, domain.ErrInactiveAccount
	}
	return s.issue(ctx, u)
}
func (s Service) issue(ctx context.Context, u domain.User) (Session, error) {
	access, e := s.Access.Issue(u.ID, []string(u.Roles), "access")
	if e != nil {
		return Session{}, e
	}
	refresh := NewRefreshToken()
	if e = s.UserRepo.SaveRefresh(ctx, u.ID, hash(refresh), time.Now().Add(s.RefreshTTL)); e != nil {
		return Session{}, e
	}
	return Session{User: u, AccessToken: access, RefreshToken: refresh, TokenType: "Bearer", ExpiresIn: int64(s.Access.ttl.Seconds())}, nil
}
func hash(v string) string {
	h := sha256.Sum256([]byte(v))
	return hex.EncodeToString(h[:])
}

func NewRefreshToken() string {
	return uuid.NewString() + uuid.NewString()
}
