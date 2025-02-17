package domain

import "errors"

var (
	ErrUserAlreadyExists = errors.New("user already exists with email: %s")
	ErrRecordNotFound    = errors.New("record not found")
	ErrEditConflict      = errors.New("edit conflict")
)
