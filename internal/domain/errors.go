package domain

import "errors"

var (
	ErrValidation      = errors.New("validation failed")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrInactiveAccount = errors.New("inactive account")
	ErrInvalidToken    = errors.New("invalid token")
	ErrExpiredToken    = errors.New("expired token")
	ErrRefreshReuse    = errors.New("refresh token reuse")
)

type FieldError struct{ Field, Message string }

func (e FieldError) Error() string { return e.Field + ": " + e.Message }
