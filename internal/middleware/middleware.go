package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/jsonutil"
)

func RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				resp := api.ErrorResponse{
					Message:   "The server encountered a problem and could not process your request",
					RequestId: middleware.GetReqID(r.Context()),
					Timestamp: time.Now(),
				}

				jsonutil.WriteJSON(w, http.StatusInternalServerError, resp, http.Header{
					"Connection": []string{"close"},
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	resp := api.ErrorResponse{
		Message:   "Resource not found",
		RequestId: middleware.GetReqID(r.Context()),
		Timestamp: time.Now(),
	}

	jsonutil.WriteJSON(w, http.StatusNotFound, resp, nil)
}
