package app

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/oapi-codegen/runtime/types"
)

func (app *application) RegisterUser(w http.ResponseWriter, r *http.Request) {
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

	err = app.userRepo.Create(r.Context(), &user)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserAlreadyExists):
			app.logError(r, fmt.Errorf(err.Error(), user.Email))
			// do not return the info of existence of email to avoid user enumeration attacks
			app.badRequestResponse(w, r, fmt.Errorf("invalid input data"))
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	token, err := domain.GenerateToken(int64(user.ID), 10*time.Minute, domain.UserActivationScope)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.tokenRepo.Create(r.Context(), token)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error(fmt.Sprintf("panic occurred during sending activation mail: %v", err))
			}
		}()

		data := map[string]any{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}

		err = app.mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			app.logger.Error(err.Error())
		}
	}()

	resp := api.RegisterResponse{
		Id:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		BirthDate: types.Date{Time: user.BirthDate},
		Gender:    api.Gender(user.Gender),
		Activated: user.Activated,
		Version:   user.Version,
	}

	err = app.writeJSON(w, http.StatusAccepted, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) ActivateUser(w http.ResponseWriter, r *http.Request) {
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
		app.logger.Error(fmt.Sprintf("user with id %d is already activated", user.ID))
		app.editConflictResponse(w, r)
		return
	}

	user.Activated = true

	err = app.userRepo.Update(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	err = app.tokenRepo.DeleteAllForUser(r.Context(), domain.UserActivationScope, user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	resp := api.UserActivationResponse{Activated: true}

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
