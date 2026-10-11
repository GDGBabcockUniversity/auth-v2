package domain

import "errors"

var (
	ErrNotFound     = errors.New("record not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrConflict     = errors.New("identity conflict")
	ErrInactive     = errors.New("inactive account")
	ErrBadInput     = errors.New("invalid input")
	ErrInvalidToken = errors.New("invalid token")
)
