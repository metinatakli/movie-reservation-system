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
