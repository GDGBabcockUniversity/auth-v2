package dto

type LoginRequest struct {
	FirebaseToken string `json:"firebase_token"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ProfileUpdateRequest map[string]any
type AdminUpdateRequest map[string]any
