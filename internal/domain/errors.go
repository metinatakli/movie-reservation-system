package domain

import "errors"

var (
	ErrUserAlreadyExists   = errors.New("user already exists with email: %s")
	ErrRecordNotFound      = errors.New("record not found")
	ErrEditConflict        = errors.New("edit conflict")
	ErrSeatAlreadyReserved = errors.New("seat(s) are already reserved")
	ErrCartNotFound        = errors.New("cart not found or has expired")
	ErrSeatLockExpired     = errors.New("your selections have expired, please select your seats again")
	ErrSeatConflict        = errors.New("a selected seat does not belong to the current session")
)
