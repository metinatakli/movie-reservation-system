package app

import (
	"errors"
	"net/http"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/oapi-codegen/runtime/types"
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
