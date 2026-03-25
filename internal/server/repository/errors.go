package repository

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrSecretNotFound     = errors.New("secret not found")
	ErrSecretDeleted      = errors.New("secret is deleted")
	ErrVersionMismatch    = errors.New("version mismatch")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
