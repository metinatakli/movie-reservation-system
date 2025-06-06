package domain

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Gender string

const (
	Male   Gender = "M"
	Female Gender = "F"
	Other  Gender = "OTHER"
)

type User struct {
	ID        int
	FirstName string
	LastName  string
	Email     string
	Password  password
	BirthDate time.Time
	Gender    Gender
	CreatedAt time.Time
	UpdatedAt time.Time
	Activated bool
	IsActive  bool
	Version   int
}

type password struct {
	plaintext *string
	Hash      []byte
}

func (p *password) Set(plaintext string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintext
	p.Hash = hash

	return nil
}

func (p *password) Matches(plaintext string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.Hash, []byte(plaintext))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

type UserRepository interface {
	CreateWithToken(context.Context, *User, func(*User) (*Token, error)) (*Token, error)
	GetByToken(ctx context.Context, tokenHash []byte, tokenScope string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetById(ctx context.Context, id int) (*User, error)
	Update(context.Context, *User) error
	ActivateUser(context.Context, *User) error
	Delete(ctx context.Context, user *User) error
}
