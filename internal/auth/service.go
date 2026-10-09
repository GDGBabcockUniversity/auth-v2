package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/google/uuid"
)

type Service struct {
	UserRepo        domain.UserRepository
	Google          *GoogleProvider
	Access, Refresh *Manager
	RefreshTTL      time.Duration
}

func (s Service) googleUser(ctx context.Context, code string) (domain.User, error) {
	if s.Google == nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	identity, err := s.Google.IdentityFromCode(ctx, code)
	if err != nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	user, err := s.UserRepo.FindOrCreateGoogleUser(ctx, identity)
	if err != nil {
		return domain.User{}, err
	}
	if !user.Active {
		return domain.User{}, domain.ErrInactiveAccount
	}
	return user, nil
}

func (s Service) LoginWithGoogleCode(ctx context.Context, code string) (Session, error) {
	user, err := s.googleUser(ctx, code)
	if err != nil {
		return Session{}, err
	}
	return s.issue(ctx, user)
}

// CreateGoogleHandoff stores only a hash of a short-lived, single-use code.
func (s Service) CreateGoogleHandoff(ctx context.Context, code string) (string, error) {
	user, err := s.googleUser(ctx, code)
	if err != nil {
		return "", err
	}
	repo, ok := s.UserRepo.(domain.GoogleHandoffRepository)
	if !ok {
		return "", errors.New("google handoff storage is unavailable")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	handoff := base64.RawURLEncoding.EncodeToString(raw)
	if err := repo.CreateGoogleHandoff(ctx, user.ID, hash(handoff), time.Now().UTC().Add(2*time.Minute)); err != nil {
		return "", err
	}
	return handoff, nil
}

func (s Service) ExchangeGoogleHandoff(ctx context.Context, handoff string) (Session, error) {
	raw, err := base64.RawURLEncoding.DecodeString(handoff)
	if err != nil || len(raw) != 32 {
		return Session{}, domain.ErrUnauthorized
	}
	repo, ok := s.UserRepo.(domain.GoogleHandoffRepository)
	if !ok {
		return Session{}, errors.New("google handoff storage is unavailable")
	}
	user, err := repo.ConsumeGoogleHandoff(ctx, hash(handoff))
	if err != nil {
		return Session{}, err
	}
	if !user.Active {
		return Session{}, domain.ErrInactiveAccount
	}
	return s.issue(ctx, user)
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
	return Session{
		User:         u,
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.Access.ttl.Seconds()),
	}, nil
}

func hash(v string) string {
	h := sha256.Sum256([]byte(v))
	return hex.EncodeToString(h[:])
}

func NewRefreshToken() string {
	return uuid.NewString() + uuid.NewString()
}
