package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"
)

func (app *Application) RegisterUser(w http.ResponseWriter, r *http.Request) {
	logger := app.contextGetLogger(r)

	var input api.RegisterRequest

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	err = app.validator.Struct(input)
	if err != nil {
		app.failedValidationResponse(w, r, err)
		return
	}

	user := domain.User{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Email:     input.Email,
		BirthDate: input.BirthDate.Time,
		Gender:    domain.Gender(input.Gender),
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	token, err := app.userRepo.CreateWithToken(r.Context(), &user, func(user *domain.User) (*domain.Token, error) {
		return domain.GenerateToken(int64(user.ID), 10*time.Minute, domain.UserActivationScope)
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserAlreadyExists):
			logger.Warn("registration attempt for existing email")
			// do not return the info of existence of email to avoid user enumeration attacks
			app.badRequestResponse(w, r, fmt.Errorf("invalid input data"))
		default:
			logger.Error("failed to create user", "error", err)
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	go func(ctx context.Context) {
		// new logger for this goroutine, inheriting context from the request
		// important for tracing across async boundaries
		gLogger := app.contextGetLogger(r.WithContext(ctx))

		defer func() {
			if err := recover(); err != nil {
				gLogger.Error("panic occurred during sending activation mail", "panic", r)
			}
		}()

		data := map[string]any{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}

		err = app.mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			gLogger.Error("failed to send activation email", "error", err)
		} else {
			gLogger.Info("activation email sent successfully")
		}
	}(r.Context())

	resp := api.UserResponse{
		Id:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		BirthDate: types.Date{Time: user.BirthDate},
		Gender:    api.Gender(user.Gender),
		Activated: user.Activated,
		CreatedAt: user.CreatedAt,
		Version:   user.Version,
	}

	err = app.writeJSON(w, http.StatusAccepted, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *Application) ActivateUser(w http.ResponseWriter, r *http.Request) {
	logger := app.contextGetLogger(r)

	var input api.UserActivationRequest

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	err = app.validator.Struct(input)
	if err != nil {
		app.failedValidationResponse(w, r, err)
		return
	}

	hash := sha256.Sum256([]byte(input.Token))
	user, err := app.userRepo.GetByToken(r.Context(), hash[:], domain.UserActivationScope)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	if user.Activated {
		logger.Warn("attempt to activate already activated user")
		app.editConflictResponse(w, r)
		return
	}

	err = app.userRepo.ActivateUser(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	resp := api.UserActivationResponse{Activated: true}

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) Login(w http.ResponseWriter, r *http.Request) {
	logger := app.contextGetLogger(r)

	userId := app.sessionManager.GetInt(r.Context(), SessionKeyUserId.String())
	if userId != 0 {
		resp := api.AlreadyLoggedInResponse{
			Message: "You are already logged in",
		}

		err := app.writeJSON(w, http.StatusOK, resp, nil)
		if err != nil {
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	var input api.LoginRequest

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	err = app.validator.Struct(input)
	if err != nil {
		logger.Warn("login validation failed")
		app.invalidCredentialsResponse(w, r)
		return
	}

	user, err := app.userRepo.GetByEmail(r.Context(), input.Email)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRecordNotFound):
			logger.Warn("login attempt for non-existent user")
			app.invalidCredentialsResponse(w, r)
		default:
			logger.Error("failed to get user by email during login", "error", err)
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	err = bcrypt.CompareHashAndPassword(user.Password.Hash, []byte(input.Password))
	if err != nil {
		logger.Warn("login failed due to incorrect password")
		app.invalidCredentialsResponse(w, r)
		return
	}

	oldSessionId := app.sessionManager.Token(r.Context())

	// To help prevent session fixation attacks we should renew the session token after any privilege level change.
	// https://github.com/OWASP/CheatSheetSeries/blob/master/cheatsheets/Session_Management_Cheat_Sheet.md#renew-the-session-id-after-any-privilege-level-change
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	newSessionId := app.sessionManager.Token(r.Context())
	err = app.migrateSessionData(r.Context(), oldSessionId, newSessionId)
	if err != nil {
		logger.Error(
			"failed to migrate session data",
			"error", err,
			"oldSessionId", oldSessionId,
			"newSessionId", newSessionId,
		)
	}

	app.sessionManager.Put(r.Context(), SessionKeyUserId.String(), user.ID)

	w.WriteHeader(http.StatusNoContent)
}

func (app *Application) Logout(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), SessionKeyUserId.String())
	if userId == 0 {
		app.notFoundResponse(w, r)
		return
	}

	app.sessionManager.Destroy(r.Context())

	w.WriteHeader(http.StatusNoContent)
}
