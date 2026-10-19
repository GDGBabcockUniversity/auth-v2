package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/GDGBabcockUniversity/auth-v2/internal/httpapi"
)

func TestUnverifiedEmailCannotTakeExistingAccount(t *testing.T) {
	store := &memoryStore{users: map[string]domain.User{
		"existing-id": {ID: "existing-id", FirebaseUID: "original", Email: "person@example.com", IsActive: true},
	}, hashes: map[string]bool{}}
	svc := service(identityVerifier{identity: auth.Identity{UID: "different-uid", Email: "person@example.com"}}, store)
	_, pair, err := svc.Login(context.Background(), "firebase-id-token", nil, nil)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected identity conflict, got %v", err)
	}
	if store.created != 0 || store.users["existing-id"].FirebaseUID != "original" || pair.AccessToken != "" {
		t.Fatal("conflicting identity created an account, changed its owner, or received a session")
	}
}

func TestEmailLoginAllowsSignupAndSigninWithoutVerification(t *testing.T) {
	for _, tc := range []struct {
		name     string
		verified bool
	}{
		{"unverified", false}, {"verified", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &memoryStore{users: map[string]domain.User{}, hashes: map[string]bool{}}
			svc := service(identityVerifier{identity: auth.Identity{UID: "email-user", Email: "person@example.com", EmailVerified: tc.verified}}, store)
			app := (httpapi.API{Service: svc}).App()
			for _, step := range []string{"signup", "signin"} {
				req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"firebase_token":"firebase-id-token"}`))
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req)
				if err != nil {
					t.Fatal(err)
				}
				var body struct {
					Error  string `json:"error"`
					Tokens struct {
						AccessToken  string `json:"access_token"`
						RefreshToken string `json:"refresh_token"`
					} `json:"tokens"`
				}
				err = json.NewDecoder(resp.Body).Decode(&body)
				resp.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if resp.StatusCode != 200 {
					t.Fatalf("%s: status=%d error=%q", step, resp.StatusCode, body.Error)
				}
				if body.Tokens.AccessToken == "" || body.Tokens.RefreshToken == "" {
					t.Fatalf("%s: missing session tokens", step)
				}
				if store.created != 1 {
					t.Fatalf("%s: created %d users", step, store.created)
				}
				if store.users["new-uuid"].EmailVerified != tc.verified {
					t.Fatalf("%s: incorrect stored verification status", step)
				}
			}
		})
	}
}
