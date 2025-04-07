package domain

import "github.com/stripe/stripe-go/v82"

type PaymentProvider interface {
	CreateCheckoutSession(sessionId string, user *User, cart Cart) (*stripe.CheckoutSession, error)
}
