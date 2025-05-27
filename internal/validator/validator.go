package validator

import (
	"fmt"
	"reflect"
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

const (
	ErrRequired        = "is required"
	ErrInvalidEmail    = "must be a valid email address"
	ErrMinLength       = "must be at least %s characters long"
	ErrMaxLength       = "must be at most %s characters long"
	ErrMinValue        = "must be at least %s"
	ErrMaxValue        = "must be at most %s"
	ErrArrayMinLength  = "must contain at least %s items"
	ErrArrayMaxLength  = "must contain at most %s items"
	ErrOnlyLetters     = "must contain only letters"
	ErrAgeCheck        = "must be at least 15 years old"
	ErrDefaultInvalid  = "is invalid"
	ErrInvalidPassword = "must be at least 8 characters long and include at least one uppercase letter, one lowercase letter, " +
		"one number, and one special character (!@#$%^&*)."
	ErrOneOf = "must be one of %s"
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
		return ErrRequired
	case "email":
		return ErrInvalidEmail
	case "min":
		switch err.Kind() {
		case reflect.String:
			return fmt.Sprintf(ErrMinLength, err.Param())
		case reflect.Slice, reflect.Array:
			return fmt.Sprintf("must contain at least %s items", err.Param())
		default:
			return fmt.Sprintf(ErrMinValue, err.Param())
		}
	case "max":
		switch err.Kind() {
		case reflect.String:
			return fmt.Sprintf(ErrMaxLength, err.Param())
		case reflect.Slice, reflect.Array:
			return fmt.Sprintf("must contain at most %s items", err.Param())
		default:
			return fmt.Sprintf(ErrMaxValue, err.Param())
		}
	case "alpha":
		return ErrOnlyLetters
	case "age_check":
		return ErrAgeCheck
	case "password":
		return ErrInvalidPassword
	case "oneof":
		return fmt.Sprintf(ErrOneOf, err.Param())
	default:
		return ErrDefaultInvalid
	}
}
