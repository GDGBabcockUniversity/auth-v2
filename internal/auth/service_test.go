package auth_test

import (
	"context"
	"errors"
	"strings"
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
	claimed int
}

func (m *memoryStore) FindByFirebaseUID(_ context.Context, uid string) (domain.User, error) {
	for _, u := range m.users {
		if u.FirebaseUID == uid {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}
func (m *memoryStore) ExistsByEmail(_ context.Context, email string) (bool, error) {
	for _, u := range m.users {
		if u.Email == email {
			return true, nil
		}
	}
	return false, nil
}
func (m *memoryStore) FindByID(_ context.Context, id string) (domain.User, error) {
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
func (m *memoryStore) UpdateLastLogin(context.Context, string) error { return nil }
func (m *memoryStore) CreateRefreshToken(_ context.Context, t *domain.RefreshToken) error {
	m.hashes[t.TokenHash] = true
	return nil
}
func (m *memoryStore) ExistsActiveRefreshToken(_ context.Context, hash, userID string) (bool, error) {
	return m.hashes[hash], nil
}
func (m *memoryStore) RevokeRefreshTokens(_ context.Context, userID, hash string) error {
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
func (m *memoryStore) FindByEmail(_ context.Context, email string) (domain.User, error) {
	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}
func (m *memoryStore) ClaimPendingUser(_ context.Context, id string, updates map[string]any) (domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	if uid, ok := updates["firebase_uid"].(string); ok {
		u.FirebaseUID = uid
	}
	if name, ok := updates["full_name"].(string); ok {
		u.FullName = name
	}
	if verified, ok := updates["email_verified"].(bool); ok {
		u.EmailVerified = verified
	}
	m.users[id] = u
	m.claimed++
	return u, nil
}
func (m *memoryStore) SetUserDeleted(_ context.Context, _, id string, deleted bool) (domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	if u.Deleted() == deleted {
		return domain.User{}, domain.ErrBadInput
	}
	if deleted {
		now := time.Now().UTC()
		u.DeletedAt, u.IsActive = &now, false
	} else {
		u.DeletedAt, u.IsActive = nil, true
	}
	m.users[id] = u
	return u, nil
}
func (m *memoryStore) FindUsers(context.Context, domain.UserQuery) ([]domain.User, int64, error) {
	return nil, 0, nil
}
func (m *memoryStore) UpdateUser(context.Context, string, string, map[string]any) (domain.User, error) {
	return domain.User{}, nil
}
func (m *memoryStore) CreateAuditEvent(context.Context, domain.AuditEvent) error { return nil }
func service(v identityVerifier, m *memoryStore) auth.Service {
	return auth.Service{Verifier: v, Repository: m, Tokens: auth.Tokens{Secret: []byte("01234567890123456789012345678901"), AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour}}
}
func TestLoginPreservesExistingUUIDAndRefreshSession(t *testing.T) {
	ctx := context.Background()
	m := &memoryStore{users: map[string]domain.User{"stable-id": {ID: "stable-id", FirebaseUID: "firebase-1", Email: "person@example.com", IsActive: true, Roles: []string{"admin"}}}, hashes: map[string]bool{}}
	s := service(identityVerifier{identity: auth.Identity{UID: "firebase-1", Email: "person@example.com", EmailVerified: true}}, m)
	u, pair, err := s.Login(ctx, "firebase-id-token", nil, nil)
	if err != nil || u.ID != "stable-id" || m.created != 0 {
		t.Fatalf("login changed identity: user=%+v err=%v", u, err)
	}
	if _, _, err = s.AccessUser(ctx, pair.RefreshToken); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("refresh token accepted as access: %v", err)
	}
	refreshed, err := s.Refresh(ctx, pair.RefreshToken)
	if err != nil || refreshed.RefreshToken != pair.RefreshToken {
		t.Fatalf("refresh broke stored session: %v", err)
	}
	if err = s.Logout(ctx, u.ID, pair.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Refresh(ctx, pair.RefreshToken); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("revoked refresh accepted: %v", err)
	}
}
func TestNewUIDCannotTakeExistingEmail(t *testing.T) {
	m := &memoryStore{users: map[string]domain.User{"stable-id": {ID: "stable-id", FirebaseUID: "original", Email: "person@example.com", IsActive: true}}, hashes: map[string]bool{}}
	s := service(identityVerifier{identity: auth.Identity{UID: "attacker", Email: "person@example.com", EmailVerified: true}}, m)
	if _, _, err := s.Login(context.Background(), "other-firebase-token", nil, nil); !errors.Is(err, domain.ErrConflict) {
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
	if _, _, err = s.AccessUser(context.Background(), pair.AccessToken); !errors.Is(err, domain.ErrInactive) {
		t.Fatalf("disabled access accepted: %v", err)
	}
	if _, _, err = s.Login(context.Background(), "token", nil, nil); !errors.Is(err, domain.ErrInactive) {
		t.Fatalf("disabled login accepted: %v", err)
	}
}

func TestLoginClaimsPreRegisteredRowAndKeepsItsRoles(t *testing.T) {
	ctx := context.Background()
	waiting := domain.User{
		ID: "pre-assigned-id", FirebaseUID: domain.PendingUID(),
		Email: "newcomer@example.com", FullName: "Newcomer",
		Roles: []string{"user", "lead"}, Teams: []string{"core"}, IsActive: true,
	}
	m := &memoryStore{users: map[string]domain.User{waiting.ID: waiting}, hashes: map[string]bool{}}
	s := service(identityVerifier{identity: auth.Identity{
		UID: "firebase-new", Email: "newcomer@example.com", Name: "New Comer", EmailVerified: true,
	}}, m)

	u, _, err := s.Login(ctx, "firebase-id-token", nil, nil)
	if err != nil {
		t.Fatalf("claiming a pre-registered row failed: %v", err)
	}
	if u.ID != waiting.ID {
		t.Fatalf("login created a second row: id = %q, want %q", u.ID, waiting.ID)
	}
	if m.created != 0 || m.claimed != 1 {
		t.Fatalf("created = %d, claimed = %d; want 0 and 1", m.created, m.claimed)
	}
	if u.FirebaseUID != "firebase-new" || u.Pending() {
		t.Fatalf("row still pending after login: %+v", u)
	}
	if len(u.Roles) != 2 || u.Roles[1] != "lead" || len(u.Teams) != 1 {
		t.Fatalf("pre-assigned roles or teams lost: roles=%v teams=%v", u.Roles, u.Teams)
	}
}

func TestLoginStillRejectsEmailHeldByAnotherFirebaseUID(t *testing.T) {
	ctx := context.Background()
	taken := domain.User{ID: "taken", FirebaseUID: "firebase-old", Email: "person@example.com", IsActive: true}
	m := &memoryStore{users: map[string]domain.User{taken.ID: taken}, hashes: map[string]bool{}}
	s := service(identityVerifier{identity: auth.Identity{UID: "firebase-different", Email: "person@example.com"}}, m)

	if _, _, err := s.Login(ctx, "firebase-id-token", nil, nil); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict for a claimed email, got %v", err)
	}
}

func TestCreateUserRejectsDuplicateEmailAndDefaultsRoles(t *testing.T) {
	ctx := context.Background()
	m := &memoryStore{users: map[string]domain.User{}, hashes: map[string]bool{}}
	s := service(identityVerifier{}, m)

	user, err := s.CreateUser(ctx, "admin-id", "  Fresh@Example.COM ", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "fresh@example.com" {
		t.Fatalf("email not normalised: %q", user.Email)
	}
	if user.FullName != "fresh" {
		t.Fatalf("full name not derived from email: %q", user.FullName)
	}
	if len(user.Roles) != 1 || user.Roles[0] != "user" || !user.Pending() {
		t.Fatalf("unexpected defaults: roles=%v pending=%v", user.Roles, user.Pending())
	}

	if _, err := s.CreateUser(ctx, "admin-id", "fresh@example.com", "", nil, nil); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate email accepted: %v", err)
	}
	if _, err := s.CreateUser(ctx, "admin-id", "not-an-email", "", nil, nil); !errors.Is(err, domain.ErrBadInput) {
		t.Fatalf("malformed email accepted: %v", err)
	}
}

func TestArchiveRevokesTokensAndRefusesSelf(t *testing.T) {
	ctx := context.Background()
	target := domain.User{ID: "target", FirebaseUID: "f1", Email: "t@example.com", IsActive: true}
	m := &memoryStore{users: map[string]domain.User{target.ID: target}, hashes: map[string]bool{"live-token": true}}
	s := service(identityVerifier{}, m)

	if _, err := s.SetUserArchived(ctx, "admin-id", "admin-id", true); !errors.Is(err, domain.ErrBadInput) {
		t.Fatalf("admin archived themselves: %v", err)
	}

	archived, err := s.SetUserArchived(ctx, "admin-id", target.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !archived.Deleted() || archived.IsActive {
		t.Fatalf("archive left the account usable: %+v", archived)
	}
	if len(m.hashes) != 0 {
		t.Fatalf("refresh tokens survived the archive: %v", m.hashes)
	}

	restored, err := s.SetUserArchived(ctx, "admin-id", target.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Deleted() || !restored.IsActive {
		t.Fatalf("restore did not reactivate: %+v", restored)
	}
}

func TestUpdateUserAcceptsProfileFieldsButNotSystemColumns(t *testing.T) {
	ctx := context.Background()
	m := &memoryStore{users: map[string]domain.User{}, hashes: map[string]bool{}}
	s := service(identityVerifier{}, m)

	if _, err := s.UpdateUser(ctx, "admin", "target", map[string]any{"department": " Computer Science "}); err != nil {
		t.Fatalf("admin could not correct a department: %v", err)
	}
	// tos_agreed is a consent record; an admin must not tick it for someone else.
	for _, blocked := range []string{"email", "firebase_uid", "id", "deleted_at", "tos_agreed"} {
		if _, err := s.UpdateUser(ctx, "admin", "target", map[string]any{blocked: "x"}); !errors.Is(err, domain.ErrBadInput) {
			t.Fatalf("%s was editable through the admin update: %v", blocked, err)
		}
	}
}
