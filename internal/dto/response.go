package dto

type ErrorResponse struct {
	Message string `json:"message"`
}

type UserResponse struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	FullName      string   `json:"full_name"`
	Roles         []string `json:"roles"`
	EmailVerified bool     `json:"email_verified"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type SessionResponse struct {
	Success bool          `json:"success"`
	User    UserResponse  `json:"user"`
	Tokens  TokenResponse `json:"tokens"`
}
