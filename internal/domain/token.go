package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"
)

const (
	UserActivationScope string = "user_activation"
	tokenLength         int    = 32
)

type Token struct {
	Plaintext string
	Hash      []byte
	UserId    int64
	Expiry    time.Time
	Scope     string
}

func GenerateToken(userId int64, ttl time.Duration, scope string) (*Token, error) {
	randomBytes := make([]byte, tokenLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	plaintext := base64.RawURLEncoding.EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(plaintext))

	token := &Token{
		Plaintext: plaintext,
		Hash:      hash[:],
		UserId:    userId,
		Expiry:    time.Now().Add(ttl),
		Scope:     scope,
	}

	return token, nil
}

type TokenRepository interface {
	Create(context.Context, *Token) error
	DeleteAllForUser(ctx context.Context, tokenScope string, userID int) error
}
