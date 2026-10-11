package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
	"uuid"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

const issuer = "auth-service"
const audience = "platform-services"

type Tokens struct {
	Secret                []byte
	AccessTTL, RefreshTTL time.Duration
}
type Claims struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles,omitempty"`
	Teams  []string `json:"teams,omitempty"`
	Type   string   `json:"token_use,omitempty"`
	jwt.RegisteredClaims
}

func (t Tokens) Issue(u domain.User, kind string) (string, error) {
	ttl := t.AccessTTL
	if kind == "refresh" {
		ttl = t.RefreshTTL
	}
	now := time.Now().UTC()

	c := Claims{
		UserID: u.ID,
		Type:   kind,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.New().String(),
		},
	}
	if kind == "access" {
		c.Roles = []string(u.Roles)
		c.Teams = []string(u.Teams)
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(t.Secret)
}

func (t Tokens) Verify(raw, kind string) (Claims, error) {
	var c Claims
	token, e := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(issuer), jwt.WithAudience(audience), jwt.WithExpirationRequired()).ParseWithClaims(raw, &c, func(_ *jwt.Token) (any, error) { return t.Secret, nil })

	if e != nil || !token.Valid || c.UserID == "" {
		return Claims{}, domain.ErrInvalidToken
	}
	if c.Type != "" && c.Type != kind {
		return Claims{}, domain.ErrInvalidToken
	}
	if c.Type == "" {
		if kind == "access" && c.Roles == nil {
			return Claims{}, domain.ErrInvalidToken
		}
		if kind == "refresh" && c.Roles != nil {
			return Claims{}, domain.ErrInvalidToken
		}
	}
	return c, nil
}

func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
