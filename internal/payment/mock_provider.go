package payment

import (
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/stripe/stripe-go/v82"
)

type MockPaymentProvider struct {
	CheckoutSession *stripe.CheckoutSession
	Err             error
}

func NewMockPaymentProvider() *MockPaymentProvider {
	return &MockPaymentProvider{}
}

func (m *MockPaymentProvider) CreateCheckoutSession(
	sessionId string,
	user *domain.User,
	cart domain.Cart,
	payment domain.Payment) (*stripe.CheckoutSession, error) {

	return m.CheckoutSession, m.Err
}
