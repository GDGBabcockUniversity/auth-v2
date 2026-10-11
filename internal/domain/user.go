package domain

import (
	"errors"
	"github.com/lib/pq"
	"time"
)

var ErrNotFound = errors.New("record not found")

type User struct {
	ID                  string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	FirebaseUID         string         `gorm:"column:firebase_uid;uniqueIndex" json:"firebase_uid"`
	Email               string         `gorm:"uniqueIndex" json:"email"`
	FullName            string         `json:"full_name"`
	WhatsAppNumber      *string        `gorm:"column:whatsapp_number" json:"whatsapp_number,omitempty"`
	AvatarURL           *string        `gorm:"column:avatar_url" json:"avatar_url,omitempty"`
	EmailVerified       bool           `gorm:"column:email_verified" json:"email_verified"`
	Gender              *string        `json:"gender,omitempty"`
	Birthday            *string        `json:"birthday,omitempty"`
	StudentStatus       *string        `json:"student_status,omitempty"`
	MatricNo            *string        `json:"matric_no,omitempty"`
	Department          *string        `json:"department,omitempty"`
	Faculty             *string        `json:"faculty,omitempty"`
	PrimaryTrack        *string        `json:"primary_track,omitempty"`
	SecondaryTrack      *string        `json:"secondary_track,omitempty"`
	PrimarySkillLevel   *string        `json:"primary_skill_level,omitempty"`
	SecondarySkillLevel *string        `json:"secondary_skill_level,omitempty"`
	Teams               pq.StringArray `gorm:"type:text[]" json:"teams"`
	Roles               pq.StringArray `gorm:"type:text[]" json:"roles"`
	TOSAgreed           bool           `gorm:"column:tos_agreed" json:"tos_agreed"`
	TOSAgreedAt         *time.Time     `gorm:"column:tos_agreed_at" json:"tos_agreed_at,omitempty"`
	TOSVersion          *string        `gorm:"column:tos_version" json:"tos_version,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	LastLoginAt         *time.Time     `gorm:"column:last_login_at" json:"last_login_at,omitempty"`
	IsActive            bool           `gorm:"column:is_active" json:"is_active"`
	DeletedAt           *time.Time     `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (User) TableName() string { return "users" }

type RefreshToken struct {
	ID        string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string     `gorm:"column:user_id;type:uuid"`
	TokenHash string     `gorm:"column:token_hash"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	RevokedAt *time.Time `gorm:"column:revoked_at"`
	IsActive  bool       `gorm:"column:is_active"`
	IPAddress *string    `gorm:"column:ip_address;type:inet"`
	UserAgent *string    `gorm:"column:user_agent"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

type AuditEvent struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    *string   `gorm:"column:user_id;type:uuid"`
	EventType string    `gorm:"column:event_type"`
	EventData []byte    `gorm:"column:event_data;type:jsonb"`
	IPAddress *string   `gorm:"column:ip_address;type:inet"`
	UserAgent *string   `gorm:"column:user_agent"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (AuditEvent) TableName() string { return "auth_audit_log" }

type Session struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         string    `gorm:"column:user_id;type:uuid"`
	SessionToken   string    `gorm:"column:session_token"`
	ExpiresAt      time.Time `gorm:"column:expires_at"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	LastActivityAt time.Time `gorm:"column:last_activity_at"`
	DeviceInfo     []byte    `gorm:"column:device_info;type:jsonb"`
}

func (Session) TableName() string { return "user_sessions" }
