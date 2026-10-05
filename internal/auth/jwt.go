package auth

import (
	"errors"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	Roles     []string `json:"roles"`
	TokenType string   `json:"token_type"`
	jwt.RegisteredClaims
}
type Manager struct {
	secret    []byte
	issuer    string
	audiences []string
	ttl       time.Duration
}

func NewManager(secret, issuer string, audiences []string, ttl time.Duration) *Manager {
	return &Manager{[]byte(secret), issuer, audiences, ttl}
}

func (m *Manager) Issue(userID string, roles []string, kind string) (string, error) {
	now := time.Now().UTC()
	c := Claims{
		Roles:     roles,
		TokenType: kind,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   m.issuer,
			Subject:  userID,
			Audience: m.audiences,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			ID:        uuid.NewString(),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}
func (m *Manager) Verify(raw, expectedAudience, kind string) (*Claims, error) {
	c := &Claims{}
	t, err := jwt.ParseWithClaims(raw, c, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, domain.ErrInvalidToken
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithAudience(expectedAudience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !t.Valid {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domain.ErrExpiredToken
		}
		return nil, domain.ErrInvalidToken
	}
	if c.TokenType != kind {
		return nil, domain.ErrInvalidToken
	}
	return c, nil
}
