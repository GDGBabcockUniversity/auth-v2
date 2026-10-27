package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GDGBabcockUniversity/auth-v2/internal/httpapi"
)

func TestDashboardServesEmbeddedAssets(t *testing.T) {
	app := httpapi.API{Origins: []string{"http://localhost:3000"}}.App()

	// Both spellings serve the page, and its assets load by absolute path either way.
	for path, want := range map[string]string{
		"/dashboard":            "<title>Chapter register",
		"/dashboard/":           "<title>Chapter register",
		"/dashboard/styles.css": "--rail-w",
		"/dashboard/app.js":     "/auth/login",
	} {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("%s: status %d", path, resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), want) {
			t.Fatalf("%s: body missing %q", path, want)
		}
	}
}

func TestDashboardConfigReportsFirebaseReadiness(t *testing.T) {
	decode := func(api httpapi.API) (bool, httpapi.FirebaseWebConfig) {
		t.Helper()
		resp, err := api.App().Test(httptest.NewRequest(http.MethodGet, "/dashboard/config.json", nil))
		if err != nil {
			t.Fatal(err)
		}
		var payload struct {
			Firebase   httpapi.FirebaseWebConfig `json:"firebase"`
			Configured bool                      `json:"configured"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		return payload.Configured, payload.Firebase
	}

	if configured, _ := decode(httpapi.API{Origins: []string{"http://localhost:3000"}}); configured {
		t.Fatal("empty Firebase config reported as ready")
	}

	want := httpapi.FirebaseWebConfig{APIKey: "web-key", AuthDomain: "gdg.firebaseapp.com", ProjectID: "gdg"}
	configured, got := decode(httpapi.API{Origins: []string{"http://localhost:3000"}, Firebase: want})
	if !configured {
		t.Fatal("complete Firebase config reported as not ready")
	}
	if got != want {
		t.Fatalf("config.json returned %+v, want %+v", got, want)
	}
}
