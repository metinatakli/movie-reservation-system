package app

import (
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

	// TODO: send the token via email

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
