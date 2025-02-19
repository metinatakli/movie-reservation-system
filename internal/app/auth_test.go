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

	"github.com/alexedwards/scs/v2"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/metinatakli/movie-reservation-system/internal/mocks"
	"github.com/metinatakli/movie-reservation-system/internal/validator"
	"github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"
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
			wantErrMessage: validator.ErrDefaultInvalid,
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
			app := newTestApplication(func(a *application) {
				a.userRepo = &mocks.MockUserRepo{CreateFunc: tt.userRepoFunc}
				a.tokenRepo = &mocks.MockTokenRepo{CreateFunc: tt.tokenRepoFunc}
				a.mailer = &MockMailer{sendFunc: tt.mailerFunc}
			})

			w, r := executeRequest(t, http.MethodPost, "/users", tt.input)

			app.RegisterUser(w, r)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("RegisterUser() status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusAccepted {
				var response api.RegisterResponse
				err := json.NewDecoder(w.Body).Decode(&response)
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
			}

			checkErrorResponse(t, w, struct {
				wantStatus     int
				wantErrMessage string
			}{
				wantStatus:     tt.wantStatus,
				wantErrMessage: tt.wantErrMessage,
			})
		})
	}
}

func TestActivateUser(t *testing.T) {
	tests := []struct {
		name               string
		input              api.UserActivationRequest
		getUserByTokenFunc func(context.Context, []byte, string) (*domain.User, error)
		updateUserFunc     func(context.Context, *domain.User) error
		deleteTokenFunc    func(context.Context, string, int) error
		wantStatus         int
		wantErrMessage     string
	}{
		{
			name: "successful activation",
			input: api.UserActivationRequest{
				Token: "O8N3AqxZYwWDq2pXWZXM4yqpyoXKUYXzV5bV0z5dL5k",
			},
			getUserByTokenFunc: func(ctx context.Context, hash []byte, scope string) (*domain.User, error) {
				return &domain.User{ID: 1, Activated: false}, nil
			},
			updateUserFunc: func(ctx context.Context, u *domain.User) error {
				return nil
			},
			deleteTokenFunc: func(ctx context.Context, scope string, userID int) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid token",
			input: api.UserActivationRequest{
				Token: "invalid-token",
			},
			getUserByTokenFunc: func(ctx context.Context, hash []byte, scope string) (*domain.User, error) {
				return nil, domain.ErrRecordNotFound
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: validator.ErrDefaultInvalid,
		},
		{
			name: "already activated user",
			input: api.UserActivationRequest{
				Token: "O8N3AqxZYwWDq2pXWZXM4yqpyoXKUYXzV5bV0z5dL5k",
			},
			getUserByTokenFunc: func(ctx context.Context, hash []byte, scope string) (*domain.User, error) {
				return &domain.User{ID: 1, Activated: true}, nil
			},
			wantStatus:     http.StatusConflict,
			wantErrMessage: ErrEditConflict,
		},
		{
			name: "update conflict",
			input: api.UserActivationRequest{
				Token: "O8N3AqxZYwWDq2pXWZXM4yqpyoXKUYXzV5bV0z5dL5k",
			},
			getUserByTokenFunc: func(ctx context.Context, hash []byte, scope string) (*domain.User, error) {
				return &domain.User{ID: 1, Activated: false}, nil
			},
			updateUserFunc: func(ctx context.Context, u *domain.User) error {
				return domain.ErrEditConflict
			},
			wantStatus:     http.StatusConflict,
			wantErrMessage: ErrEditConflict,
		},
		{
			name: "token deletion failure",
			input: api.UserActivationRequest{
				Token: "O8N3AqxZYwWDq2pXWZXM4yqpyoXKUYXzV5bV0z5dL5k",
			},
			getUserByTokenFunc: func(ctx context.Context, hash []byte, scope string) (*domain.User, error) {
				return &domain.User{ID: 1, Activated: false}, nil
			},
			updateUserFunc: func(ctx context.Context, u *domain.User) error {
				return nil
			},
			deleteTokenFunc: func(ctx context.Context, scope string, userID int) error {
				return fmt.Errorf("failed to delete token")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(func(a *application) {
				a.userRepo = &mocks.MockUserRepo{
					GetByTokenFunc: tt.getUserByTokenFunc,
					UpdateFunc:     tt.updateUserFunc,
				}
				a.tokenRepo = &mocks.MockTokenRepo{
					DeleteAllForUserFunc: tt.deleteTokenFunc,
				}
			})

			w, r := executeRequest(t, http.MethodPut, "/users/activation", tt.input)

			app.ActivateUser(w, r)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("ActivateUser() status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var response api.UserActivationResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if !response.Activated {
					t.Error("Expected Activated=true in response")
				}
			}

			checkErrorResponse(t, w, struct {
				wantStatus     int
				wantErrMessage string
			}{
				wantStatus:     tt.wantStatus,
				wantErrMessage: tt.wantErrMessage,
			})
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name           string
		input          api.LoginRequest
		getByEmailFunc func(context.Context, string) (*domain.User, error)
		password       string
		wantStatus     int
		wantErrMessage string
	}{
		{
			name: "successful login",
			input: api.LoginRequest{
				Email:    "freddie@example.com",
				Password: "Pass123!@#",
			},
			password: "Pass123!@#",
			getByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Pass123!@#"), 12)
				user := &domain.User{}

				user.ID = 1
				user.Password.Hash = hashedPassword

				return user, nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "invalid password format",
			input: api.LoginRequest{
				Email:    "freddie@example.com",
				Password: "weak",
			},
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: ErrInvalidCredentials,
		},
		{
			name: "user not found",
			input: api.LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "Pass123!@#",
			},
			getByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
				return nil, domain.ErrRecordNotFound
			},
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: ErrInvalidCredentials,
		},
		{
			name: "incorrect password",
			input: api.LoginRequest{
				Email:    "freddie@example.com",
				Password: "WrongPass123!@#",
			},
			password: "Pass123!@#",
			getByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Pass12!@#"), 12)
				user := &domain.User{}

				user.ID = 1
				user.Password.Hash = hashedPassword

				return user, nil
			},
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: ErrInvalidCredentials,
		},
		{
			name: "database error",
			input: api.LoginRequest{
				Email:    "freddie@example.com",
				Password: "Pass123!@#",
			},
			getByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
				return nil, fmt.Errorf("database connection error")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(func(a *application) {
				a.userRepo = &mocks.MockUserRepo{
					GetByEmailFunc: tt.getByEmailFunc,
				}
				a.sessionManager = scs.New()
			})

			w, r := executeRequest(t, http.MethodPost, "/sessions", tt.input)

			handler := app.sessionManager.LoadAndSave(http.HandlerFunc(app.Login))
			handler.ServeHTTP(w, r)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("Login() status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusNoContent {
				var sessionCookie *http.Cookie
				for _, cookie := range w.Result().Cookies() {
					if cookie.Name == app.sessionManager.Cookie.Name {
						sessionCookie = cookie
						break
					}
				}

				if sessionCookie == nil {
					t.Fatal("No session cookie found in response")
					return
				}

				ctx, err := app.sessionManager.Load(r.Context(), sessionCookie.Value)
				if err != nil {
					t.Fatalf("Failed to load session: %v", err)
				}

				userId := app.sessionManager.GetInt(ctx, SessionKeyUserId)

				if userId != 1 {
					t.Errorf("Expected userId=1 in session, got %v", userId)
				}
			}

			checkErrorResponse(t, w, struct {
				wantStatus     int
				wantErrMessage string
			}{
				wantStatus:     tt.wantStatus,
				wantErrMessage: tt.wantErrMessage,
			})
		})
	}
}

func TestLogout(t *testing.T) {
	tests := []struct {
		name           string
		setupSession   bool
		wantStatus     int
		wantErrMessage string
	}{
		{
			name:         "successful logout",
			setupSession: true,
			wantStatus:   http.StatusNoContent,
		},
		{
			name:           "no active session",
			setupSession:   false,
			wantStatus:     http.StatusNotFound,
			wantErrMessage: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(func(a *application) {
				a.sessionManager = scs.New()
			})

			w, r := executeRequest(t, http.MethodDelete, "/sessions", nil)

			if tt.setupSession {
				r = setupTestSession(t, app, r, 1)
			}

			handler := app.sessionManager.LoadAndSave(http.HandlerFunc(app.Logout))
			handler.ServeHTTP(w, r)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("Logout() status = %v, want %v", got, tt.wantStatus)
			}

			if tt.setupSession {
				userId := app.sessionManager.GetInt(r.Context(), SessionKeyUserId)
				if userId != 0 {
					t.Error("Session was not destroyed")
				}
			}

			checkErrorResponse(t, w, struct {
				wantStatus     int
				wantErrMessage string
			}{
				wantStatus:     tt.wantStatus,
				wantErrMessage: tt.wantErrMessage,
			})
		})
	}
}

func setupTestSession(t *testing.T, app *application, r *http.Request, userId int) *http.Request {
	ctx, err := app.sessionManager.Load(r.Context(), "session")
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	app.sessionManager.Put(ctx, SessionKeyUserId, userId)

	return r.WithContext(ctx)
}

func executeRequest(t *testing.T, method, url string, body any) (*httptest.ResponseRecorder, *http.Request) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(method, url, bytes.NewReader(jsonData))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	return w, r
}

func checkErrorResponse(t *testing.T, w *httptest.ResponseRecorder, tt struct {
	wantStatus     int
	wantErrMessage string
}) {
	if got := w.Code; got != tt.wantStatus {
		t.Errorf("status = %v, want %v", got, tt.wantStatus)
	}

	if tt.wantStatus >= 200 && tt.wantStatus < 300 {
		return
	}

	switch tt.wantStatus {
	case http.StatusUnprocessableEntity:
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

	default:
		var errorResp api.ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&errorResp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if tt.wantErrMessage != "" && errorResp.Message != tt.wantErrMessage {
			t.Errorf("Error message = %v, want %v", errorResp.Message, tt.wantErrMessage)
		}
	}
}

func newTestApplication(opts ...func(*application)) *application {
	app := &application{
		validator: validator.NewValidator(),
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		userRepo:  &mocks.MockUserRepo{},
		tokenRepo: &mocks.MockTokenRepo{},
		mailer:    &MockMailer{},
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}
