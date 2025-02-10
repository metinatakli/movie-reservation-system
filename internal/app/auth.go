package app

import (
	"fmt"
	"net/http"

	"github.com/metinatakli/movie-reservation-system/api"
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
		app.unprocessableEntityResponse(w, r, err)
		return
	}

	fmt.Fprint(w, "im here")
}
