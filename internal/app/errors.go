package app

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/metinatakli/movie-reservation-system/api"
	appvalidator "github.com/metinatakli/movie-reservation-system/internal/validator"
)

const (
	ErrInternalServer     = "The server encountered a problem and could not process your request"
	ErrNotFound           = "The requested resource not found"
	ErrEditConflict       = "Unable to update the record due to an edit conflict, please try again"
	ErrInvalidCredentials = "Invalid email or password"
	ErrUnauthorizedAccess = "You must be authenticated to access this resource"
	ErrForbiddenAccess    = "You do not have permission to perform this action"
)

func (app *Application) logError(r *http.Request, err error) {
	var (
		method    = r.Method
		uri       = r.URL.RequestURI()
		requestId = middleware.GetReqID(r.Context())
	)

	app.logger.Error(err.Error(), "method", method, "uri", uri, "request-id", requestId)
}

// The errorResponse() method is a generic helper for sending JSON-formatted error
// messages to the client with a given status code.
func (app *Application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message string) {
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

func (app *Application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	app.errorResponse(w, r, http.StatusInternalServerError, ErrInternalServer)
}

func (app *Application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusNotFound, ErrNotFound)
}

func (app *Application) notFoundResponseWithErr(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusNotFound, err.Error())
}

func (app *Application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (app *Application) failedValidationResponse(w http.ResponseWriter, r *http.Request, err error) {
	var validationErrs []api.ValidationError

	for _, err := range err.(validator.ValidationErrors) {
		validationErrs = append(validationErrs, api.ValidationError{
			Field: err.StructField(),
			Issue: appvalidator.ValidationMessage(err),
		})
	}

	resp := api.ValidationErrorResponse{
		Message:          "One or more fields have invalid values",
		RequestId:        middleware.GetReqID(r.Context()),
		Timestamp:        time.Now(),
		ValidationErrors: validationErrs,
	}

	err = app.writeJSON(w, http.StatusUnprocessableEntity, resp, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(500)
	}
}

func (app *Application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusConflict, ErrEditConflict)
}

func (app *Application) editConflictResponseWithErr(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusConflict, err.Error())
}

func (app *Application) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusUnauthorized, ErrInvalidCredentials)
}

func (app *Application) unauthorizedAccessResponse(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusUnauthorized, ErrUnauthorizedAccess)
}

func (app *Application) forbiddenResponse(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusForbidden, ErrForbiddenAccess)
}
