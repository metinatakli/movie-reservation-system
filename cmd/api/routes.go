package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/metinatakli/movie-reservation-system/api"
	app_mw "github.com/metinatakli/movie-reservation-system/internal/middleware"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.NotFound(app_mw.NotFoundHandler)

	r.Use(middleware.Logger)
	r.Use(middleware.RequestID)
	r.Use(app_mw.RecoverPanic)

	return api.HandlerFromMux(app.handlers, r)
}
