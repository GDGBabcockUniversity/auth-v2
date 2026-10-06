package auth

import (
	"context"
	"testing"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/lib/pq"
)

type fakeUsers struct {
	user    domain.User
	refresh map[string]string
}

func (f *fakeUsers) Create(_ context.Context, email, name, hash string) (domain.User, error) {
	if f.user.ID != "" {
		return domain.User{}, domain.ErrConflict
	}
	f.user = domain.User{ID: "u1", Email: email, FullName: name, PasswordHash: hash, Roles: pq.StringArray{"user"}, Active: true}
	return f.user, nil
}
func (f *fakeUsers) FindOrCreateGoogleUser(_ context.Context, identity domain.GoogleIdentity) (domain.User, error) {
	if f.user.ID != "" {
		return f.user, nil
	}
	f.user = domain.User{ID: "u1", Email: identity.Email, FullName: identity.Name, GoogleSubject: identity.Subject, Roles: pq.StringArray{"user"}, Active: true, EmailVerified: true}
	return f.user, nil
}
func (f *fakeUsers) FindByEmail(_ context.Context, email string) (domain.User, error) {
	if f.user.Email != email {
		return domain.User{}, domain.ErrNotFound
	}
	return f.user, nil
}
func (f *fakeUsers) FindByID(_ context.Context, id string) (domain.User, error) {
	if f.user.ID != id {
		return domain.User{}, domain.ErrNotFound
	}
	return f.user, nil
}
func (f *fakeUsers) SaveRefresh(_ context.Context, id, hash string, _ time.Time) error {
	if f.refresh == nil {
		f.refresh = map[string]string{}
	}
	f.refresh[hash] = id
	return nil
}
func (f *fakeUsers) ConsumeRefresh(_ context.Context, hash string) (domain.User, error) {
	id, ok := f.refresh[hash]
	if !ok {
		return domain.User{}, domain.ErrRefreshReuse
	}
	delete(f.refresh, hash)
	return f.FindByID(context.Background(), id)
}

func TestServiceRegistersLogsInAndRotatesRefresh(t *testing.T) {
	repo := &fakeUsers{}
	m := NewManager("12345678901234567890123456789012", "https://auth.gdgbabcock.com", []string{"https://gdgbabcock.com"}, time.Hour)
	s := Service{UserRepo: repo, Access: m, Refresh: m, RefreshTTL: time.Hour}
	created, e := s.Register(context.Background(), "Member@Example.com", "Member", "a secure password")
	if e != nil {
		t.Fatal(e)
	}
	if created.User.Email != "member@example.com" || created.AccessToken == "" {
		t.Fatalf("bad registration: %#v", created)
	}
	logged, e := s.Login(context.Background(), "member@example.com", "a secure password")
	if e != nil || logged.RefreshToken == "" {
		t.Fatalf("login: %v", e)
	}
	rotated, e := s.RefreshSession(context.Background(), logged.RefreshToken)
	if e != nil || rotated.RefreshToken == logged.RefreshToken {
		t.Fatalf("rotation: %v", e)
	}
	if _, e = s.RefreshSession(context.Background(), logged.RefreshToken); e != domain.ErrRefreshReuse {
		t.Fatalf("reuse should fail: %v", e)
	}
}
