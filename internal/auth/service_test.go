package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
)

type identityVerifier struct {
	identity auth.Identity
	err      error
}

func (v identityVerifier) Verify(context.Context, string) (auth.Identity, error) {
	return v.identity, v.err
}

type memoryStore struct {
	users   map[string]domain.User
	hashes  map[string]bool
	created int
}

func (m *memoryStore) ByUID(_ context.Context, uid string) (domain.User, error) {
	for _, u := range m.users {
		if u.FirebaseUID == uid {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}
func (m *memoryStore) ByEmail(_ context.Context, email string) (domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}
func (m *memoryStore) ByID(_ context.Context, id string) (domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return u, domain.ErrNotFound
	}
	return u, nil
}
func (m *memoryStore) Create(_ context.Context, u *domain.User) (bool, error) {
	m.created++
	u.ID = "new-uuid"
	m.users[u.ID] = *u
	return true, nil
}
func (m *memoryStore) LastLogin(context.Context, string) error { return nil }
func (m *memoryStore) StoreRefresh(_ context.Context, t *domain.RefreshToken) error {
	m.hashes[t.TokenHash] = true
	return nil
}
func (m *memoryStore) ActiveRefresh(_ context.Context, hash, userID string) (bool, error) {
	return m.hashes[hash], nil
}
func (m *memoryStore) RevokeRefresh(_ context.Context, userID, hash string) error {
	if hash == "" {
		clear(m.hashes)
	} else {
		delete(m.hashes, hash)
	}
	return nil
}
func (m *memoryStore) UpdateProfile(context.Context, string, map[string]any) (domain.User, error) {
	return domain.User{}, nil
}
func (m *memoryStore) AdminList(context.Context, string, string, int, int) ([]domain.User, int64, error) {
	return nil, 0, nil
}
func (m *memoryStore) AdminUpdate(context.Context, string, string, map[string]any) (domain.User, error) {
	return domain.User{}, nil
}
func (m *memoryStore) Audit(context.Context, domain.AuditEvent) error { return nil }
func service(v identityVerifier, m *memoryStore) auth.Service {
	return auth.Service{Verifier: v, Store: m, Tokens: auth.Tokens{Secret: []byte("01234567890123456789012345678901"), AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour}}
}
func TestLoginPreservesExistingUUIDAndRefreshSession(t *testing.T) {
	ctx := context.Background()
	m := &memoryStore{users: map[string]domain.User{"stable-id": {ID: "stable-id", FirebaseUID: "firebase-1", Email: "person@example.com", IsActive: true, Roles: []string{"admin"}}}, hashes: map[string]bool{}}
	s := service(identityVerifier{identity: auth.Identity{UID: "firebase-1", Email: "person@example.com", EmailVerified: true}}, m)
	u, pair, err := s.Login(ctx, "firebase-id-token", nil, nil)
	if err != nil || u.ID != "stable-id" || m.created != 0 {
		t.Fatalf("login changed identity: user=%+v err=%v", u, err)
	}
	if _, _, err = s.AccessUser(ctx, pair.RefreshToken); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("refresh token accepted as access: %v", err)
	}
	refreshed, err := s.Refresh(ctx, pair.RefreshToken)
	if err != nil || refreshed.RefreshToken != pair.RefreshToken {
		t.Fatalf("refresh broke stored session: %v", err)
	}
	if err = s.Logout(ctx, u.ID, pair.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Refresh(ctx, pair.RefreshToken); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("revoked refresh accepted: %v", err)
	}
}
func TestNewUIDCannotTakeExistingEmail(t *testing.T) {
	m := &memoryStore{users: map[string]domain.User{"stable-id": {ID: "stable-id", FirebaseUID: "original", Email: "person@example.com", IsActive: true}}, hashes: map[string]bool{}}
	s := service(identityVerifier{identity: auth.Identity{UID: "attacker", Email: "person@example.com", EmailVerified: true}}, m)
	if _, _, err := s.Login(context.Background(), "other-firebase-token", nil, nil); !errors.Is(err, auth.ErrConflict) {
		t.Fatalf("wanted identity conflict, got %v", err)
	}
	if m.users["stable-id"].FirebaseUID != "original" || m.created != 0 {
		t.Fatal("existing identity was modified")
	}
}
func TestInactiveUserCannotLoginOrUseExistingAccess(t *testing.T) {
	m := &memoryStore{users: map[string]domain.User{"stable-id": {ID: "stable-id", FirebaseUID: "original", Email: "person@example.com", IsActive: true, Roles: []string{"user"}}}, hashes: map[string]bool{}}
	s := service(identityVerifier{identity: auth.Identity{UID: "original", Email: "person@example.com"}}, m)
	_, pair, err := s.Login(context.Background(), "token", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	u := m.users["stable-id"]
	u.IsActive = false
	m.users[u.ID] = u
	if _, _, err = s.AccessUser(context.Background(), pair.AccessToken); !errors.Is(err, auth.ErrInactive) {
		t.Fatalf("disabled access accepted: %v", err)
	}
	if _, _, err = s.Login(context.Background(), "token", nil, nil); !errors.Is(err, auth.ErrInactive) {
		t.Fatalf("disabled login accepted: %v", err)
	}
}
