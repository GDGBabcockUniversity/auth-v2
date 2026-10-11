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

// Maps preserve field presence for partial updates and allow unknown fields to
// be rejected by the service rather than silently discarded by JSON binding.
type ProfileUpdateRequest map[string]any
type AdminUpdateRequest map[string]any
