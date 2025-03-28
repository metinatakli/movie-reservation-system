package domain

import "errors"

var (
	ErrUserAlreadyExists   = errors.New("user already exists with email: %s")
	ErrRecordNotFound      = errors.New("record not found")
	ErrEditConflict        = errors.New("edit conflict")
	ErrSeatAlreadyReserved = errors.New("seat(s) are already reserved")
)
