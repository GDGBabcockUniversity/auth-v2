package dto

import (
	"time"

	"github.com/GDGBabcockUniversity/auth-v2/internal/auth"
)

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type LoginUser struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	FullName  string   `json:"full_name"`
	AvatarURL *string  `json:"avatar_url"`
	Roles     []string `json:"roles"`
	Teams     []string `json:"teams"`
}

type ProfileUser struct {
	ID                  string     `json:"id"`
	FirebaseUID         string     `json:"firebase_uid"`
	Email               string     `json:"email"`
	FullName            string     `json:"full_name"`
	WhatsAppNumber      *string    `json:"whatsapp_number"`
	AvatarURL           *string    `json:"avatar_url"`
	EmailVerified       bool       `json:"email_verified"`
	Gender              *string    `json:"gender"`
	Birthday            *string    `json:"birthday"`
	StudentStatus       *string    `json:"student_status"`
	MatricNo            *string    `json:"matric_no"`
	Department          *string    `json:"department"`
	Faculty             *string    `json:"faculty"`
	PrimaryTrack        *string    `json:"primary_track"`
	SecondaryTrack      *string    `json:"secondary_track"`
	PrimarySkillLevel   *string    `json:"primary_skill_level"`
	SecondarySkillLevel *string    `json:"secondary_skill_level"`
	Teams               []string   `json:"teams"`
	Roles               []string   `json:"roles"`
	TOSAgreed           bool       `json:"tos_agreed"`
	TOSAgreedAt         *time.Time `json:"tos_agreed_at"`
	TOSVersion          *string    `json:"tos_version"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	LastLoginAt         *time.Time `json:"last_login_at"`
	IsActive            bool       `json:"is_active"`
}

type AdminUser struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	AvatarURL   *string    `json:"avatar_url"`
	Roles       []string   `json:"roles"`
	Teams       []string   `json:"teams"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

type LoginResponse struct {
	Success bool      `json:"success"`
	User    LoginUser `json:"user"`
	Tokens  TokenPair `json:"tokens"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type TokensResponse struct {
	Success bool      `json:"success"`
	Tokens  TokenPair `json:"tokens"`
}

type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ProfileResponse struct {
	Success bool        `json:"success"`
	User    ProfileUser `json:"user"`
}

type VerifyResponse struct {
	Success bool        `json:"success"`
	Valid   bool        `json:"valid"`
	User    auth.Claims `json:"user"`
}

type AdminListResponse struct {
	Success bool        `json:"success"`
	Users   []AdminUser `json:"users"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	Limit   int         `json:"limit"`
}

type AdminUserResponse struct {
	Success bool      `json:"success"`
	User    AdminUser `json:"user"`
}
