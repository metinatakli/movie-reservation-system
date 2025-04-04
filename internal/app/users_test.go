package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/metinatakli/movie-reservation-system/internal/mocks"
	"github.com/metinatakli/movie-reservation-system/internal/validator"
	"github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"
)

func TestGetUsersMe(t *testing.T) {
	tests := []struct {
		name           string
		setupSession   bool
		userId         int
		getByIdFunc    func(context.Context, int) (*domain.User, error)
		wantStatus     int
		wantErrMessage string
		wantResponse   *api.UserResponse
	}{
		{
			name:         "successful retrieval",
			setupSession: true,
			userId:       1,
			getByIdFunc: func(ctx context.Context, id int) (*domain.User, error) {
				return &domain.User{
					ID:        1,
					FirstName: "Freddie",
					LastName:  "Mercury",
					Email:     "freddie@example.com",
					BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
					Gender:    "M",
					Activated: true,
					Version:   1,
					CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				}, nil
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.UserResponse{
				Id:        1,
				FirstName: "Freddie",
				LastName:  "Mercury",
				Email:     "freddie@example.com",
				BirthDate: types.Date{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)},
				Gender:    api.M,
				Activated: true,
				Version:   1,
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:           "no session",
			setupSession:   false,
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: ErrUnauthorizedAccess,
		},
		{
			name:         "user not found",
			setupSession: true,
			userId:       1,
			getByIdFunc: func(ctx context.Context, id int) (*domain.User, error) {
				return nil, domain.ErrRecordNotFound
			},
			wantStatus:     http.StatusNotFound,
			wantErrMessage: ErrNotFound,
		},
		{
			name:         "database error",
			setupSession: true,
			userId:       1,
			getByIdFunc: func(ctx context.Context, id int) (*domain.User, error) {
				return nil, fmt.Errorf("database error")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(func(a *application) {
				a.userRepo = &mocks.MockUserRepo{
					GetByIdFunc: tt.getByIdFunc,
				}
				a.sessionManager = scs.New()
			})

			w, r := executeRequest(t, http.MethodGet, "/users/me", nil)

			if tt.setupSession {
				r = setupTestSession(t, app, r, tt.userId)
			}

			handler := app.requireAuthentication(http.HandlerFunc(app.GetCurrentUser))
			handler = app.sessionManager.LoadAndSave(handler)
			handler.ServeHTTP(w, r)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantResponse != nil {
				var response api.UserResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if diff := cmp.Diff(tt.wantResponse, &response); diff != "" {
					t.Errorf("Mismatch (-want +got):\n%s", diff)
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

func TestUpdateUser(t *testing.T) {
	tests := []struct {
		name           string
		setupSession   bool
		userId         int
		input          api.UpdateUserRequest
		getByIdFunc    func(context.Context, int) (*domain.User, error)
		updateFunc     func(context.Context, *domain.User) error
		wantStatus     int
		wantErrMessage string
		wantResponse   *api.UserResponse
	}{
		{
			name:         "successful update",
			setupSession: true,
			userId:       1,
			input: api.UpdateUserRequest{
				FirstName: ptr("Freddy"),
				LastName:  ptr("Mercury"),
				BirthDate: &types.Date{Time: time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC)},
				Gender:    (*api.Gender)(ptr("M")),
			},
			getByIdFunc: func(ctx context.Context, id int) (*domain.User, error) {
				return &domain.User{
					ID:        1,
					FirstName: "Freddy",
					LastName:  "Mercury",
					Email:     "freddie@example.com",
					BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
					Gender:    "M",
					Activated: true,
					Version:   1,
					CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				}, nil
			},
			updateFunc: func(ctx context.Context, user *domain.User) error {
				return nil
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.UserResponse{
				Id:        1,
				FirstName: "Freddy",
				LastName:  "Mercury",
				Email:     "freddie@example.com",
				BirthDate: types.Date{Time: time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC)},
				Gender:    api.M,
				Activated: true,
				Version:   1,
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:           "no session",
			setupSession:   false,
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: ErrUnauthorizedAccess,
		},
		{
			name:         "user not found",
			setupSession: true,
			userId:       1,
			input: api.UpdateUserRequest{
				FirstName: ptr("Freddy"),
			},
			getByIdFunc: func(ctx context.Context, id int) (*domain.User, error) {
				return nil, domain.ErrRecordNotFound
			},
			wantStatus:     http.StatusNotFound,
			wantErrMessage: ErrNotFound,
		},
		{
			name:         "edit conflict",
			setupSession: true,
			userId:       1,
			input: api.UpdateUserRequest{
				FirstName: ptr("Freddy"),
			},
			getByIdFunc: func(ctx context.Context, id int) (*domain.User, error) {
				return &domain.User{ID: 1, FirstName: "Metin"}, nil
			},
			updateFunc: func(ctx context.Context, user *domain.User) error {
				return domain.ErrEditConflict
			},
			wantStatus:     http.StatusConflict,
			wantErrMessage: ErrEditConflict,
		},
		{
			name:         "invalid first name - too short, min length is 2",
			setupSession: true,
			userId:       1,
			input: api.UpdateUserRequest{
				FirstName: ptr("A"),
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: fmt.Sprintf(validator.ErrMinLength, "2"),
		},
		{
			name:         "invalid first name - non-alpha",
			setupSession: true,
			userId:       1,
			input: api.UpdateUserRequest{
				FirstName: ptr("Freddie123"),
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: validator.ErrOnlyLetters,
		},
		{
			name:         "invalid last name - too long, max length is 50",
			setupSession: true,
			userId:       1,
			input: api.UpdateUserRequest{
				LastName: ptr("ThisIsAReallyLongLastNameThatExceedsFiftyCharactersAndShouldFail"),
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: fmt.Sprintf(validator.ErrMaxLength, "50"),
		},
		{
			name:         "invalid birth date - too young, must be at least 15 years old",
			setupSession: true,
			userId:       1,
			input: api.UpdateUserRequest{
				BirthDate: &types.Date{Time: time.Now().AddDate(-14, 0, 0)},
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: validator.ErrAgeCheck,
		},
		{
			name:         "invalid gender - must be F, M, or OTHER",
			setupSession: true,
			userId:       1,
			input: api.UpdateUserRequest{
				Gender: (*api.Gender)(ptr("INVALID")),
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: validator.ErrDefaultInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(func(a *application) {
				a.userRepo = &mocks.MockUserRepo{
					GetByIdFunc: tt.getByIdFunc,
					UpdateFunc:  tt.updateFunc,
				}
				a.sessionManager = scs.New()
			})

			w, r := executeRequest(t, http.MethodPatch, "/users/me", tt.input)

			if tt.setupSession {
				r = setupTestSession(t, app, r, tt.userId)
			}

			handler := app.requireAuthentication(http.HandlerFunc(app.UpdateUser))
			handler = app.sessionManager.LoadAndSave(handler)
			handler.ServeHTTP(w, r)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantResponse != nil {
				var response api.UserResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if diff := cmp.Diff(tt.wantResponse, &response); diff != "" {
					t.Errorf("Mismatch (-want +got):\n%s", diff)
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

func TestInitiateUserDeletion(t *testing.T) {
	tests := []struct {
		name            string
		setupSession    bool
		userId          int
		input           api.InitiateUserDeletionRequest
		getByIdFunc     func(context.Context, int) (*domain.User, error)
		createTokenFunc func(context.Context, *domain.Token) error
		wantStatus      int
		wantErrMessage  string
	}{
		{
			name:         "successful deletion initiation",
			setupSession: true,
			userId:       1,
			input: api.InitiateUserDeletionRequest{
				Password: "Correct@Pass123",
			},
			getByIdFunc: func(ctx context.Context, id int) (*domain.User, error) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Correct@Pass123"), 12)

				user := &domain.User{}
				user.Password.Hash = hashedPassword
				user.Email = "test@example.com"

				return user, nil
			},
			createTokenFunc: func(ctx context.Context, token *domain.Token) error {
				return nil
			},
			wantStatus: http.StatusAccepted,
		},
		{
			name:           "no session",
			setupSession:   false,
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: ErrUnauthorizedAccess,
		},
		{
			name:         "invalid password format",
			setupSession: true,
			userId:       1,
			input: api.InitiateUserDeletionRequest{
				Password: "weak",
			},
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: ErrInvalidCredentials,
		},
		{
			name:         "user not found",
			setupSession: true,
			userId:       1,
			input: api.InitiateUserDeletionRequest{
				Password: "Password123!",
			},
			getByIdFunc: func(ctx context.Context, id int) (*domain.User, error) {
				return nil, domain.ErrRecordNotFound
			},
			wantStatus:     http.StatusNotFound,
			wantErrMessage: ErrNotFound,
		},
		{
			name:         "incorrect password",
			setupSession: true,
			userId:       1,
			input: api.InitiateUserDeletionRequest{
				Password: "Wrong@Pass123",
			},
			getByIdFunc: func(ctx context.Context, id int) (*domain.User, error) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Correct@Pass123"), 12)

				user := &domain.User{}
				user.Password.Hash = hashedPassword
				user.Email = "test@example.com"

				return user, nil
			},
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: ErrInvalidCredentials,
		},
		{
			name:         "token creation error",
			setupSession: true,
			userId:       1,
			input: api.InitiateUserDeletionRequest{
				Password: "Correct@Pass123",
			},
			getByIdFunc: func(ctx context.Context, id int) (*domain.User, error) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Correct@Pass123"), 12)

				user := &domain.User{}
				user.Password.Hash = hashedPassword
				user.Email = "test@example.com"

				return user, nil
			},
			createTokenFunc: func(ctx context.Context, token *domain.Token) error {
				return fmt.Errorf("token creation error")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(func(a *application) {
				a.userRepo = &mocks.MockUserRepo{
					GetByIdFunc: tt.getByIdFunc,
				}
				a.tokenRepo = &mocks.MockTokenRepo{
					CreateFunc: tt.createTokenFunc,
				}
				a.sessionManager = scs.New()
			})

			w, r := executeRequest(t, http.MethodPost, "/users/me/deletion", tt.input)

			if tt.setupSession {
				r = setupTestSession(t, app, r, tt.userId)
			}

			handler := app.requireAuthentication(http.HandlerFunc(app.InitiateUserDeletion))
			handler = app.sessionManager.LoadAndSave(handler)
			handler.ServeHTTP(w, r)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("status = %v, want %v", got, tt.wantStatus)
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

func TestCompleteUserDeletion(t *testing.T) {
	tests := []struct {
		name               string
		setupSession       bool
		userId             int
		input              api.CompleteUserDeletionRequest
		getByTokenFunc     func(context.Context, []byte, string) (*domain.User, error)
		deleteFunc         func(context.Context, *domain.User) error
		deleteAllTokenFunc func(context.Context, string, int) error
		wantStatus         int
		wantErrMessage     string
	}{
		{
			name:         "successful deletion",
			setupSession: true,
			userId:       1,
			input: api.CompleteUserDeletionRequest{
				Token: "O8N3AqxZYwWDq2pXWZXM4yqpyoXKUYXzV5bV0z5dL5k",
			},
			getByTokenFunc: func(ctx context.Context, hash []byte, scope string) (*domain.User, error) {
				return &domain.User{ID: 1}, nil
			},
			deleteFunc: func(ctx context.Context, user *domain.User) error {
				return nil
			},
			deleteAllTokenFunc: func(ctx context.Context, scope string, userId int) error {
				return nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:           "no session",
			setupSession:   false,
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: ErrUnauthorizedAccess,
		},
		{
			name:         "invalid token format",
			setupSession: true,
			userId:       1,
			input: api.CompleteUserDeletionRequest{
				Token: "invalid-token",
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: validator.ErrDefaultInvalid,
		},
		{
			name:         "token not found",
			setupSession: true,
			userId:       1,
			input: api.CompleteUserDeletionRequest{
				Token: "ZY4xzVx0_qm4XQlO5P6YyGvZz9kGvYyoUn8WF3mHAGQ",
			},
			getByTokenFunc: func(ctx context.Context, hash []byte, scope string) (*domain.User, error) {
				return nil, domain.ErrRecordNotFound
			},
			wantStatus:     http.StatusNotFound,
			wantErrMessage: ErrNotFound,
		},
		{
			name:         "unauthorized deletion attempt",
			setupSession: true,
			userId:       1,
			input: api.CompleteUserDeletionRequest{
				Token: "O8N3AqxZYwWDq2pXWZXM4yqpyoXKUYXzV5bV0z5dL5k",
			},
			getByTokenFunc: func(ctx context.Context, hash []byte, scope string) (*domain.User, error) {
				return &domain.User{ID: 2}, nil
			},
			wantStatus:     http.StatusForbidden,
			wantErrMessage: ErrForbiddenAccess,
		},
		{
			name:         "edit conflict during deletion",
			setupSession: true,
			userId:       1,
			input: api.CompleteUserDeletionRequest{
				Token: "O8N3AqxZYwWDq2pXWZXM4yqpyoXKUYXzV5bV0z5dL5k",
			},
			getByTokenFunc: func(ctx context.Context, hash []byte, scope string) (*domain.User, error) {
				return &domain.User{ID: 1}, nil
			},
			deleteFunc: func(ctx context.Context, user *domain.User) error {
				return domain.ErrEditConflict
			},
			wantStatus:     http.StatusConflict,
			wantErrMessage: ErrEditConflict,
		},
		{
			name:         "database error during user deletion",
			setupSession: true,
			userId:       1,
			input: api.CompleteUserDeletionRequest{
				Token: "O8N3AqxZYwWDq2pXWZXM4yqpyoXKUYXzV5bV0z5dL5k",
			},
			getByTokenFunc: func(ctx context.Context, hash []byte, scope string) (*domain.User, error) {
				return &domain.User{ID: 1}, nil
			},
			deleteFunc: func(ctx context.Context, user *domain.User) error {
				return fmt.Errorf("database error")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name:         "database error during token lookup",
			setupSession: true,
			userId:       1,
			input: api.CompleteUserDeletionRequest{
				Token: "O8N3AqxZYwWDq2pXWZXM4yqpyoXKUYXzV5bV0z5dL5k",
			},
			getByTokenFunc: func(ctx context.Context, hash []byte, scope string) (*domain.User, error) {
				return nil, fmt.Errorf("database error")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(func(a *application) {
				a.userRepo = &mocks.MockUserRepo{
					GetByTokenFunc: tt.getByTokenFunc,
					DeleteFunc:     tt.deleteFunc,
				}
				a.tokenRepo = &mocks.MockTokenRepo{
					DeleteAllForUserFunc: tt.deleteAllTokenFunc,
				}
				a.sessionManager = scs.New()
			})

			w, r := executeRequest(t, http.MethodPut, "/users/me/deletion-request", tt.input)

			if tt.setupSession {
				r = setupTestSession(t, app, r, tt.userId)
			}

			handler := app.requireAuthentication(http.HandlerFunc(app.CompleteUserDeletion))
			handler = app.sessionManager.LoadAndSave(handler)
			handler.ServeHTTP(w, r)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("status = %v, want %v", got, tt.wantStatus)
			}

			if http.StatusNoContent == w.Code {
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
