package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/GDGBabcockUniversity/auth-v2/internal/httpapi"
	"github.com/lib/pq"
)

type userStore struct {
	domain.AuthRepository
	user domain.User
}

func (s *userStore) FindByID(context.Context, string) (domain.User, error) { return s.user, nil }

func TestProtectedRoutesUseCurrentRolesAndRejectRefreshToken(t *testing.T) {
	user := domain.User{ID: "user-id", Email: "user@example.invalid", FullName: "Test", Roles: pq.StringArray{"admin"}, IsActive: true}
	store := &userStore{user: user}
	tokens := auth.Tokens{Secret: []byte("01234567890123456789012345678901"), AccessTTL: time.Hour, RefreshTTL: time.Hour}
	api := httpapi.API{Service: auth.Service{Repository: store, Tokens: tokens}, Origins: []string{"http://localhost:3000"}}.App()
	access, err := tokens.Issue(user, "access")
	if err != nil {
		t.Fatal(err)
	}
	refresh, err := tokens.Issue(user, "refresh")
	if err != nil {
		t.Fatal(err)
	}
	request := func(path, token string) *http.Response {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := api.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	if got := request("/auth/verify", refresh).StatusCode; got != 401 {
		t.Fatalf("refresh token authorized: %d", got)
	}
	resp := request("/auth/verify", access)
	if resp.StatusCode != 200 {
		t.Fatalf("access token rejected: %d", resp.StatusCode)
	}
	var result struct {
		User auth.Claims `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.User.ExpiresAt == nil || result.User.UserID != user.ID {
		t.Fatalf("verify omitted JWT claims: %+v", result.User)
	}
	store.user.Roles = pq.StringArray{"user"}
	if got := request("/admin/users", access).StatusCode; got != 403 {
		t.Fatalf("stale admin role authorized: %d", got)
	}
	store.user.IsActive = false
	if got := request("/auth/verify", access).StatusCode; got != 403 {
		t.Fatalf("inactive account authorized: %d", got)
	}
}
