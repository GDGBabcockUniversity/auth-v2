package domain

import "errors"

var (
	ErrValidation      = errors.New("Validation Failed")
	ErrUnauthorized    = errors.New("Unauthorized")
	ErrForbidden       = errors.New("Forbidden")
	ErrNotFound        = errors.New("Not Found")
	ErrConflict        = errors.New("Conflict")
	ErrInactiveAccount = errors.New("Inactive Account")
	ErrInvalidToken    = errors.New("Invalid Token")
	ErrExpiredToken    = errors.New("Expired Token")
	ErrRefreshReuse    = errors.New("Refresh Token Reuse")
)

type FieldError struct {
	Field   string
	Message string
}

func (e FieldError) Error() string {
	return e.Field + ": " + e.Message
}
