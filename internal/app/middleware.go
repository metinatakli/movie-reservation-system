package app

import (
	"context"
	"fmt"
	"net/http"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")

				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) ensureGuestUserSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionId := app.sessionManager.Token(r.Context())

		if sessionId == "" {
			app.sessionManager.Put(r.Context(), SessionKeyGuest.String(), true)

			_, _, err := app.sessionManager.Commit(r.Context())
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId := app.sessionManager.GetInt(r.Context(), SessionKeyUserId.String())
		if userId == 0 {
			app.unauthorizedAccessResponse(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), SessionKeyUserId, userId)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
