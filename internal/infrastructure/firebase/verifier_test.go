package firebase_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GDGBabcockUniversity/auth-v2/internal/config"
	"github.com/GDGBabcockUniversity/auth-v2/internal/infrastructure/firebase"
)

func TestNewReportsMissingServiceAccountFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-service-account.json")
	_, err := firebase.New(context.Background(), config.Config{
		FirebaseProjectID:          "test-project",
		FirebaseServiceAccountPath: path,
	})
	if err == nil || !strings.Contains(err.Error(), "FIREBASE_SERVICE_ACCOUNT_PATH") {
		t.Fatalf("expected service-account path error, got %v", err)
	}
}
