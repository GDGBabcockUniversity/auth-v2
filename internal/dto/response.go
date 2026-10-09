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

type AdminUserResponse struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	FullName      string   `json:"full_name"`
	Roles         []string `json:"roles"`
	Active        bool     `json:"active"`
	EmailVerified bool     `json:"email_verified"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type ListUsersResponse struct {
	Success bool                `json:"success"`
	Users   []AdminUserResponse `json:"users"`
	Total   int64               `json:"total"`
	Limit   int                 `json:"limit"`
	Offset  int                 `json:"offset"`
}