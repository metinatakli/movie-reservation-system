package integration_test

import (
	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

const (
	// User related constants
	TestUserFirstName = "John"
	TestUserLastName  = "Doe"
	TestUserEmail     = "test@example.com"
	TestUserPassword  = "Test123!@#"
	TestUserBirthDate = "1990-01-01"
	TestUserGender    = domain.Male

	// Token related constants
	TestActivationToken = "r8zEhnVzNTZDf8WypfYBTU_FkFUm9jXnTmMrK-WuFQ8"
	TestTokenScope      = domain.UserActivationScope
)
