package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/metinatakli/movie-reservation-system/internal/mocks"
	"github.com/metinatakli/movie-reservation-system/internal/validator"
	"github.com/oapi-codegen/runtime/types"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
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
			app := newTestApplication(func(a *Application) {
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
				var response api.UserResponse
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
			app := newTestApplication(func(a *Application) {
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

// TODO: rewrite auth_test.go using testify
type LoginTestSuite struct {
	suite.Suite
	app           *Application
	redisClient   *mocks.MockRedisClient
	redisPipeline *mocks.MockTxPipeline
}

func (s *LoginTestSuite) SetupTest() {
	s.redisClient = new(mocks.MockRedisClient)
	s.redisPipeline = new(mocks.MockTxPipeline)

	s.app = newTestApplication(func(a *Application) {
		a.redis = s.redisClient
		a.sessionManager = scs.New()
	})
}

func TestLoginSuite(t *testing.T) {
	suite.Run(t, new(LoginTestSuite))
}

func (s *LoginTestSuite) TestLogin() {
	tests := []struct {
		name           string
		input          api.LoginRequest
		getByEmailFunc func(context.Context, string) (*domain.User, error)
		setupMocks     func()
		setupSession   bool
		password       string
		wantStatus     int
		wantErrMessage string
		wantResponse   *api.AlreadyLoggedInResponse
	}{
		{
			name: "user already is logged in",
			input: api.LoginRequest{
				Email:    "freddie@example.com",
				Password: "Pass123!@#",
			},
			setupSession: true,
			wantStatus:   http.StatusOK,
			wantResponse: &api.AlreadyLoggedInResponse{Message: "You are already logged in"},
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
			setupMocks: func() {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringResult(cartID, nil)).Once()
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringResult(cartDataStr, nil)).Once()
				s.redisClient.On("TTL", mock.Anything, mock.Anything).Return(redis.NewDurationResult(2*time.Minute, nil))
				s.redisClient.On("Watch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

				s.redisClient.On("TxPipeline").Return(s.redisPipeline)
				s.redisPipeline.On("Expire", mock.Anything, mock.Anything, mock.Anything).Return(redis.NewBoolResult(true, nil))
				s.redisPipeline.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(redis.NewStatusResult("OK", nil))
				s.redisPipeline.On("Exec", mock.Anything).Return([]redis.Cmder{}, nil)
			},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()
			s.app.userRepo = &mocks.MockUserRepo{
				GetByEmailFunc: tt.getByEmailFunc,
			}

			defer s.redisClient.AssertExpectations(s.T())
			defer s.redisPipeline.AssertExpectations(s.T())

			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			w, r := executeRequest(s.T(), http.MethodPost, "/sessions", tt.input)

			if tt.setupSession {
				r = setupTestSession(s.T(), s.app, r, 1)
			}

			handler := s.app.sessionManager.LoadAndSave(http.HandlerFunc(s.app.Login))
			handler.ServeHTTP(w, r)

			s.Equal(tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusNoContent {
				var sessionCookie *http.Cookie
				for _, cookie := range w.Result().Cookies() {
					if cookie.Name == s.app.sessionManager.Cookie.Name {
						sessionCookie = cookie
						break
					}
				}

				if sessionCookie == nil {
					s.T().Fatal("No session cookie found in response")
					return
				}

				ctx, err := s.app.sessionManager.Load(r.Context(), sessionCookie.Value)
				if err != nil {
					s.T().Fatalf("Failed to load session: %v", err)
				}

				userId := s.app.sessionManager.GetInt(ctx, SessionKeyUserId.String())

				if userId != 1 {
					s.T().Errorf("Expected userId=1 in session, got %v", userId)
				}
			}

			checkErrorResponse(s.T(), w, struct {
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
			app := newTestApplication(func(a *Application) {
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
				userId := app.sessionManager.GetInt(r.Context(), SessionKeyUserId.String())
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
