package app

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/metinatakli/movie-reservation-system/api"
)

func (app *application) logError(r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	app.logger.Error(err.Error(), "method", method, "uri", uri)
}

// The errorResponse() method is a generic helper for sending JSON-formatted error
// messages to the client with a given status code.
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message string) {
	resp := api.ErrorResponse{
		Message:   message,
		RequestId: middleware.GetReqID(r.Context()),
		Timestamp: time.Now(),
	}

	err := app.writeJSON(w, status, resp, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(500)
	}
}

func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)

	message := "The server encountered a problem and could not process your request"
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "The requested resource not found"
	app.errorResponse(w, r, http.StatusNotFound, message)
}
