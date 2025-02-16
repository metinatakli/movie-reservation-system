package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/metinatakli/movie-reservation-system/internal/mocks"
	"github.com/metinatakli/movie-reservation-system/internal/validator"
	"github.com/oapi-codegen/runtime/types"
)

type MockMailer struct {
	sendFunc func(recipient, template string, data any) error
}

func (m *MockMailer) Send(recipient, template string, data any) error {
	return m.sendFunc(recipient, template, data)
}

func TestRegisterUser(t *testing.T) {
	tests := []struct {
		name           string
		input          api.RegisterRequest
		userRepoFunc   func(context.Context, *domain.User) error
		tokenRepoFunc  func(context.Context, *domain.Token) error
		mailerFunc     func(recipient, template string, data any) error
		wantStatus     int
		wantErrMessage string
	}{
		{
			name: "successful registration",
			input: api.RegisterRequest{
				FirstName: "Freddie",
				LastName:  "Mercury",
				Email:     "freddie@example.com",
				Password:  "Pass123!@#",
				BirthDate: types.Date{Time: time.Now().AddDate(-20, 0, 0)},
				Gender:    api.M,
			},
			userRepoFunc: func(ctx context.Context, u *domain.User) error {
				u.ID = 1
				return nil
			},
			tokenRepoFunc: func(ctx context.Context, t *domain.Token) error {
				return nil
			},
			mailerFunc: func(recipient, template string, data any) error {
				return nil
			},
			wantStatus: http.StatusAccepted,
		},
		{
			name: "invalid password format",
			input: api.RegisterRequest{
				FirstName: "Freddie",
				LastName:  "Mercury",
				Email:     "freddie@example.com",
				Password:  "weak",
				BirthDate: types.Date{Time: time.Now().AddDate(-20, 0, 0)},
				Gender:    api.M,
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: validator.ErrInvalidPassword,
		},
		{
			name: "underage user",
			input: api.RegisterRequest{
				FirstName: "Freddie",
				LastName:  "Mercury",
				Email:     "freddie@example.com",
				Password:  "Pass123!@#",
				BirthDate: types.Date{Time: time.Now().AddDate(-14, 0, 0)},
				Gender:    api.M,
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: validator.ErrAgeCheck,
		},
		{
			name: "invalid gender",
			input: api.RegisterRequest{
				FirstName: "Freddie",
				LastName:  "Mercury",
				Email:     "freddie@example.com",
				Password:  "Pass123!@#",
				BirthDate: types.Date{Time: time.Now().AddDate(-20, 0, 0)},
				Gender:    "INVALID",
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: validator.ErrInvalidGender,
		},
		{
			name: "duplicate email",
			input: api.RegisterRequest{
				FirstName: "Freddie",
				LastName:  "Mercury",
				Email:     "existing@example.com",
				Password:  "Pass123!@#",
				BirthDate: types.Date{Time: time.Now().AddDate(-20, 0, 0)},
				Gender:    api.M,
			},
			userRepoFunc: func(ctx context.Context, u *domain.User) error {
				return domain.ErrUserAlreadyExists
			},
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "invalid input data",
		},
		{
			name: "token creation failure",
			input: api.RegisterRequest{
				FirstName: "Freddie",
				LastName:  "Mercury",
				Email:     "freddie@example.com",
				Password:  "Pass123!@#",
				BirthDate: types.Date{Time: time.Now().AddDate(-20, 0, 0)},
				Gender:    api.M,
			},
			userRepoFunc: func(ctx context.Context, u *domain.User) error {
				u.ID = 1
				return nil
			},
			tokenRepoFunc: func(ctx context.Context, t *domain.Token) error {
				return fmt.Errorf("token creation failed")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &application{
				validator: validator.NewValidator(),
				logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
				userRepo:  &mocks.MockUserRepo{CreateFunc: tt.userRepoFunc},
				tokenRepo: &mocks.MockTokenRepo{CreateFunc: tt.tokenRepoFunc},
				mailer:    &MockMailer{sendFunc: tt.mailerFunc},
			}

			jsonData, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatal(err)
			}

			r := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(jsonData))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			app.RegisterUser(w, r)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("RegisterUser() status = %v, want %v", got, tt.wantStatus)
			}

			var response api.RegisterResponse
			if tt.wantStatus == http.StatusAccepted {
				err = json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if response.Id != 1 {
					t.Errorf("Expected id=1 in response, got %v", response.Id)
				}
				if response.Email != tt.input.Email {
					t.Errorf("Expected email=%s in response, got %v", tt.input.Email, response.Email)
				}
				if response.Activated != false {
					t.Errorf("Expected Activated=false, got %v", response.Activated)
				}
			} else if tt.wantStatus == http.StatusUnprocessableEntity {
				var validationResp api.ValidationErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&validationResp); err != nil {
					t.Fatalf("Failed to decode validation error response: %v", err)
				}

				errorSet := make(map[string]bool)
				for _, vErr := range validationResp.ValidationErrors {
					errorSet[vErr.Issue] = true
				}

				if !errorSet[tt.wantErrMessage] {
					t.Errorf("Expected validation error message '%s' not found in response", tt.wantErrMessage)
				}
			} else {
				var errorResp api.ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&errorResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}

				if errorResp.Message != tt.wantErrMessage {
					t.Errorf("Error message = %v, want %v", errorResp.Message, tt.wantErrMessage)
				}
			}
		})
	}
}
