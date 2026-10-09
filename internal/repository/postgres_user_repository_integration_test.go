package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/domain"
	"github.com/GDGBabcockUniversity/auth-v2/internal/repository"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupPostgres(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("auth"),
		tcpostgres.WithUsername("auth"),
		tcpostgres.WithPassword("auth"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.RefreshToken{}))

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		_ = ctr.Terminate(ctx)
	}
	return db, cleanup
}

func seedUser(t *testing.T, repo repository.PostgresUserRepository, email, name string) domain.User {
	t.Helper()
	u, err := repo.Create(context.Background(), email, name, "hash")
	require.NoError(t, err)
	require.NotEmpty(t, u.ID)
	return u
}

func TestPostgresUserRepository_List(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()
	repo := repository.PostgresUserRepository{DB: db}
	ctx := context.Background()

	users, total, err := repo.List(ctx, 10, 0, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), total)
	require.Empty(t, users)

	u1 := seedUser(t, repo, "alice@example.com", "Alice")
	u2 := seedUser(t, repo, "bob@example.com", "Bob")
	u3 := seedUser(t, repo, "carol@example.com", "Carol")

	_, err = repo.SetActive(ctx, u2.ID, false)
	require.NoError(t, err)

	t.Run("all users ordered by created_at desc", func(t *testing.T) {
		users, total, err := repo.List(ctx, 10, 0, nil)
		require.NoError(t, err)
		require.Equal(t, int64(3), total)
		require.Len(t, users, 3)
		require.Equal(t, u3.ID, users[0].ID)
		require.Equal(t, u2.ID, users[1].ID)
		require.Equal(t, u1.ID, users[2].ID)
	})

	t.Run("pagination", func(t *testing.T) {
		page1, total, err := repo.List(ctx, 2, 0, nil)
		require.NoError(t, err)
		require.Equal(t, int64(3), total)
		require.Len(t, page1, 2)

		page2, total, err := repo.List(ctx, 2, 2, nil)
		require.NoError(t, err)
		require.Equal(t, int64(3), total)
		require.Len(t, page2, 1)
		require.Equal(t, u1.ID, page2[0].ID)
	})

	t.Run("filter active only", func(t *testing.T) {
		active := true
		users, total, err := repo.List(ctx, 10, 0, &active)
		require.NoError(t, err)
		require.Equal(t, int64(2), total)
		require.Len(t, users, 2)
		for _, u := range users {
			require.True(t, u.Active)
		}
	})

	t.Run("filter inactive only", func(t *testing.T) {
		active := false
		users, total, err := repo.List(ctx, 10, 0, &active)
		require.NoError(t, err)
		require.Equal(t, int64(1), total)
		require.Len(t, users, 1)
		require.Equal(t, u2.ID, users[0].ID)
		require.False(t, users[0].Active)
	})

	t.Run("limit clamped", func(t *testing.T) {
		users, _, err := repo.List(ctx, 0, 0, nil)
		require.NoError(t, err)
		require.Len(t, users, 3)

		users, _, err = repo.List(ctx, 500, 0, nil)
		require.NoError(t, err)
		require.Len(t, users, 3)
	})
}

func TestPostgresUserRepository_SetActive(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()
	repo := repository.PostgresUserRepository{DB: db}
	ctx := context.Background()

	u := seedUser(t, repo, "dave@example.com", "Dave")
	require.True(t, u.Active)

	require.NoError(t, repo.SaveRefresh(ctx, u.ID, "hash1", time.Now().Add(time.Hour)))
	require.NoError(t, repo.SaveRefresh(ctx, u.ID, "hash2", time.Now().Add(time.Hour)))

	t.Run("deactivate and revoke tokens", func(t *testing.T) {
		updated, err := repo.SetActive(ctx, u.ID, false)
		require.NoError(t, err)
		require.False(t, updated.Active)
		require.Equal(t, u.ID, updated.ID)

		var count int64
		err = db.Model(&domain.RefreshToken{}).
			Where("user_id = ? AND revoked_at IS NULL", u.ID).
			Count(&count).Error
		require.NoError(t, err)
		require.Equal(t, int64(0), count)

		err = db.Model(&domain.RefreshToken{}).
			Where("user_id = ?", u.ID).
			Count(&count).Error
		require.NoError(t, err)
		require.Equal(t, int64(2), count)
	})

	t.Run("reactivate", func(t *testing.T) {
		updated, err := repo.SetActive(ctx, u.ID, true)
		require.NoError(t, err)
		require.True(t, updated.Active)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.SetActive(ctx, "00000000-0000-0000-0000-000000000000", false)
		require.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestPostgresUserRepository_Delete(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()
	repo := repository.PostgresUserRepository{DB: db}
	ctx := context.Background()

	t.Run("delete user and cascade tokens", func(t *testing.T) {
		u := seedUser(t, repo, "eve@example.com", "Eve")
		require.NoError(t, repo.SaveRefresh(ctx, u.ID, "tokhash", time.Now().Add(time.Hour)))

		err := repo.Delete(ctx, u.ID)
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, u.ID)
		require.ErrorIs(t, err, domain.ErrNotFound)

		var count int64
		err = db.Model(&domain.RefreshToken{}).Where("user_id = ?", u.ID).Count(&count).Error
		require.NoError(t, err)
		require.Equal(t, int64(0), count)
	})

	t.Run("not found", func(t *testing.T) {
		err := repo.Delete(ctx, "00000000-0000-0000-0000-000000000000")
		require.ErrorIs(t, err, domain.ErrNotFound)
	})
}
