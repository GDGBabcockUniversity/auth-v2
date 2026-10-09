package dto

type RegisterRequest struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ListUsersQuery struct {
	Limit  int   `query:"limit"`
	Offset int   `query:"offset"`
	Active *bool `query:"active"`
}

type SetActiveRequest struct {
	Active bool `json:"active"`
}
