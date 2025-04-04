package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/mocks"
	"github.com/metinatakli/movie-reservation-system/internal/validator"
)

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

func setupTestSession(t *testing.T, app *application, r *http.Request, userId int) *http.Request {
	ctx, err := app.sessionManager.Load(r.Context(), "session")
	if err != nil {
		t.Errorf("Failed to load session: %v", err)
	}

	app.sessionManager.Put(ctx, SessionKeyUserId.String(), userId)

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

func ptr[T any](v T) *T {
	return &v
}
