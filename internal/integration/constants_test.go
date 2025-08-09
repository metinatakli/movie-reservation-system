package integration_test

import (
	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

const (
	// User related constants
	TestUserId        = 1
	TestUserFirstName = "John"
	TestUserLastName  = "Doe"
	TestUserEmail     = "test@example.com"
	TestUserPassword  = "Test123!@#"
	TestUserBirthDate = "1990-01-01"
	TestUserGender    = domain.Male

	// Token related constants
	TestToken      = "r8zEhnVzNTZDf8WypfYBTU_FkFUm9jXnTmMrK-WuFQ8"
	TestTokenScope = domain.UserActivationScope

	// Checkout session related constants
	TestCheckoutSessionId  = "cs_test_12345"
	TestCheckoutSessionURL = "https://checkout.stripe.com/pay/cs_test_12345"
)
