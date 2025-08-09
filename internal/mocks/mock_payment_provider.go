package mocks

import (
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v82"
)

type MockPaymentProvider struct {
	mock.Mock
	domain.PaymentProvider
}

func (m *MockPaymentProvider) CreateCheckoutSession(
	sessionId string,
	user *domain.User,
	cart domain.Cart,
	payment domain.Payment) (*stripe.CheckoutSession, error) {

	args := m.Called(sessionId, user, cart)
	return args.Get(0).(*stripe.CheckoutSession), args.Error(1)
}
