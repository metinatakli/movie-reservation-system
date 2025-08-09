package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCanceled  PaymentStatus = "canceled"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

type Payment struct {
	ID                int
	UserID            int
	CheckoutSessionId *string
	Amount            decimal.Decimal
	Currency          string
	Status            PaymentStatus
	ErrorMsg          *string
	PaymentDate       *time.Time
	CreatedAt         time.Time
	UpdatedAt         *time.Time
}

type PaymentRepository interface {
	Create(ctx context.Context, payment *Payment) error
	GetById(ctx context.Context, id int) (*Payment, error)
	UpdateStatus(ctx context.Context, checkoutSessionID string, status PaymentStatus, errMsg string) error
}
