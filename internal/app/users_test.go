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
			wantStatus:     http.StatusNotFound,
			wantErrMessage: ErrNotFound,
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

			handler := app.sessionManager.LoadAndSave(http.HandlerFunc(app.GetCurrentUser))
			handler.ServeHTTP(w, r)

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
			wantStatus:     http.StatusNotFound,
			wantErrMessage: ErrNotFound,
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

			handler := app.sessionManager.LoadAndSave(http.HandlerFunc(app.UpdateUser))
			handler.ServeHTTP(w, r)

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
