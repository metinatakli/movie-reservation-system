package integration_test

import (
	"time"

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

	// Movie related constants
	TestMovieTitle       = "Test Movie"
	TestMovieDescription = "A test movie description."
	TestMovieLanguage    = "English"
	TestMovieDuration    = 120
	TestMoviePosterUrl   = "https://example.com/poster.jpg"
	TestMovieDirector    = "Jane Doe"
	TestMovieRating      = 7.5
)

var (
	TestMovieGenres      = []string{"Action", "Drama"}
	TestMovieCast        = []string{"Actor One", "Actor Two"}
	TestMovieReleaseDate = time.Now().Truncate(24 * time.Hour).Format("2006-01-02")
)
