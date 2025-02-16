package validator

import (
	"fmt"
	"regexp"
	"time"
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/metinatakli/movie-reservation-system/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var (
	minAge        = 15
	maxAge        = 120
	hasSpecialRgx = regexp.MustCompile(`[!@#$%^&*]`)
)

func NewValidator() *validator.Validate {
	validator := validator.New(validator.WithRequiredStructEnabled())

	validator.RegisterValidation("age_check", validateBirthDate)
	validator.RegisterValidation("password", validatePassword)
	validator.RegisterValidation("gender", validateGender)

	return validator
}

func validateGender(fl validator.FieldLevel) bool {
	gender, ok := fl.Field().Interface().(api.Gender)
	if !ok {
		return false
	}

	return gender == api.F || gender == api.M || gender == api.OTHER
}

func validateBirthDate(fl validator.FieldLevel) bool {
	birthDate := fl.Field().Interface().(openapi_types.Date).Time

	today := time.Now()
	age := today.Year() - birthDate.Year()
	if today.YearDay() < birthDate.YearDay() {
		age--
	}

	return age >= minAge && age <= maxAge
}

func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 8 || len(password) > 25 {
		return false
	}

	containsUpper, containsLower, containsDigit, containsSpecial := false, false, false, false

	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			containsUpper = true
		case unicode.IsLower(ch):
			containsLower = true
		case unicode.IsDigit(ch):
			containsDigit = true
		case hasSpecialRgx.MatchString(string(ch)):
			containsSpecial = true
		}
	}

	return containsUpper && containsLower && containsDigit && containsSpecial
}

// ValidationMessage converts validator errors into readable messages
func ValidationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters long", err.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters long", err.Param())
	case "alpha":
		return "must contain only letters"
	case "age_check":
		return "must be at least 15 years old"
	case "password":
		return "must be at least 8 characters long and include at least one uppercase letter, one lowercase letter, " +
			"one number, and one special character (!@#$%^&*)."
	default:
		return "is invalid"
	}
}
