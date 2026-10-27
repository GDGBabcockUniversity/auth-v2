package postgres_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/database"
	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/GDGBabcockUniversity/auth-v2/internal/infrastructure/postgres"
	"github.com/lib/pq"
)

func TestExistingAuthSchemaSupportsIdentityAndAdminAudit(t *testing.T) {
	url := os.Getenv("AUTH_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("AUTH_TEST_DATABASE_URL is required for disposable Postgres integration test")
	}
	db, err := database.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	store := postgres.Repository{DB: tx}
	ctx := context.Background()
	user := domain.User{FirebaseUID: "test-uid", Email: "test@example.invalid", FullName: "Test Member", Roles: pq.StringArray{"user"}, Teams: pq.StringArray{}, IsActive: true, EmailVerified: true}
	created, err := store.Create(ctx, &user)
	if err != nil || !created || user.ID == "" {
		t.Fatalf("create user: created=%v id=%q err=%v", created, user.ID, err)
	}
	found, err := store.FindByFirebaseUID(ctx, "test-uid")
	if err != nil || found.ID != user.ID {
		t.Fatalf("lookup changed UUID: %+v %v", found, err)
	}
	exists, err := store.ExistsByEmail(ctx, "TEST@example.invalid")
	if err != nil || !exists {
		t.Fatalf("case-insensitive email existence: exists=%v err=%v", exists, err)
	}
	listed, total, err := store.FindUsers(ctx, domain.UserQuery{Search: "test@example.invalid", Role: "user", Page: 1, Limit: 20})
	if err != nil || total != 1 || len(listed) != 1 || listed[0].ID != user.ID {
		t.Fatalf("admin list: total=%d users=%+v err=%v", total, listed, err)
	}
	token := domain.RefreshToken{UserID: user.ID, TokenHash: "test-hash", ExpiresAt: time.Now().Add(time.Hour), IsActive: true}
	if err = store.CreateRefreshToken(ctx, &token); err != nil {
		t.Fatal(err)
	}
	active, err := store.ExistsActiveRefreshToken(ctx, "test-hash", user.ID)
	if err != nil || !active {
		t.Fatalf("stored refresh missing: %v", err)
	}
	updated, err := store.UpdateUser(ctx, user.ID, user.ID, map[string]any{"roles": pq.StringArray{"admin"}, "is_active": false})
	if err != nil || updated.IsActive || len(updated.Roles) != 1 || updated.Roles[0] != "admin" {
		t.Fatalf("admin update: %+v %v", updated, err)
	}
	var count int64
	if err = tx.Model(&domain.AuditEvent{}).Where("user_id = ? AND event_type = ?", user.ID, "user_admin_update").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("admin audit missing: count=%d err=%v", count, err)
	}
	var event domain.AuditEvent
	if err = tx.Where("user_id = ? AND event_type = ?", user.ID, "user_admin_update").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	var detail struct {
		Target string `json:"target_user_id"`
		After  struct {
			Roles    []string `json:"roles"`
			IsActive bool     `json:"is_active"`
		} `json:"after"`
	}
	if err = json.Unmarshal(event.EventData, &detail); err != nil || detail.Target != user.ID || detail.After.IsActive || len(detail.After.Roles) != 1 || detail.After.Roles[0] != "admin" {
		t.Fatalf("admin audit detail missing: %+v err=%v", detail, err)
	}
}
