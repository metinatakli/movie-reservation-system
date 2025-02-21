package app

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), SessionKeyUserId)
	if userId == 0 {
		app.notFoundResponse(w, r)
		return
	}

	user, err := app.userRepo.GetById(r.Context(), userId)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRecordNotFound):
			app.logger.Error("User ID in session but not found in DB", "userId", userId)
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	resp := api.UserResponse{
		Id:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		BirthDate: types.Date{Time: user.BirthDate},
		Gender:    api.Gender(user.Gender),
		Activated: user.Activated,
		Version:   user.Version,
		CreatedAt: user.CreatedAt,
	}

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), SessionKeyUserId)
	if userId == 0 {
		app.notFoundResponse(w, r)
		return
	}

	var input api.UpdateUserRequest

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

	user, err := app.userRepo.GetById(r.Context(), userId)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	if input.FirstName != nil {
		user.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		user.LastName = *input.LastName
	}
	if input.BirthDate != nil {
		user.BirthDate = input.BirthDate.Time
	}
	if input.Gender != nil {
		user.Gender = domain.Gender(*input.Gender)
	}

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

	resp := api.UserResponse{
		Id:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		BirthDate: types.Date{Time: user.BirthDate},
		Gender:    api.Gender(user.Gender),
		Activated: user.Activated,
		Version:   user.Version,
		CreatedAt: user.CreatedAt,
	}

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) InitiateUserDeletion(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), SessionKeyUserId)
	if userId == 0 {
		app.notFoundResponse(w, r)
		return
	}

	var input api.InitiateUserDeletionRequest

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	err = app.validator.Struct(input)
	if err != nil {
		app.invalidCredentialsResponse(w, r)
		return
	}

	user, err := app.userRepo.GetById(r.Context(), userId)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	err = bcrypt.CompareHashAndPassword(user.Password.Hash, []byte(input.Password))
	if err != nil {
		app.invalidCredentialsResponse(w, r)
		return
	}

	token, err := domain.GenerateToken(int64(userId), time.Duration(30*time.Minute), domain.UserDeletionScope)
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
				app.logger.Error(fmt.Sprintf("panic occurred during sending user deletion mail: %v", err))
			}
		}()

		data := map[string]any{
			"deletionToken": token.Plaintext,
			"userID":        user.ID,
		}

		err = app.mailer.Send(user.Email, "user_deletion.tmpl", data)
		if err != nil {
			app.logger.Error(err.Error())
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}
