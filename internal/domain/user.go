package domain

import (
	"time"

	"github.com/lib/pq"
)

type User struct {
	ID            string `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Email         string `gorm:"uniqueIndex;not null"`
	FullName      string
	PasswordHash  string
	Roles         pq.StringArray `gorm:"type:text[];not null"`
	Active        bool           `gorm:"column:is_active;default:true"`
	EmailVerified bool           `gorm:"default:false"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (User) TableName() string { return "users" }

type RefreshToken struct {
	ID        string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    string    `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
