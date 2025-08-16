package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type contextKey string

const loggerContextKey = contextKey("logger")

func (app *Application) recoverPanic(next http.Handler) http.Handler {
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

func (app *Application) ensureGuestUserSession(next http.Handler) http.Handler {
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

func (app *Application) requireAuthentication(next http.Handler) http.Handler {
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

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

// newLoggingResponseWriter creates a new loggingResponseWriter.
func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	// Default to 200 OK if WriteHeader is not explicitly called.
	return &loggingResponseWriter{w, http.StatusOK, 0}
}

// WriteHeader captures the status code before calling the underlying ResponseWriter.
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Write captures the number of bytes written before calling the underlying ResponseWriter.
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := lrw.ResponseWriter.Write(b)
	lrw.bytes += size

	return size, err
}

func (app *Application) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestLogger := app.logger.With(
			"request_id", middleware.GetReqID(r.Context()),
			"user_id", app.sessionManager.GetInt(r.Context(), SessionKeyUserId.String()),
			"session_id", app.sessionManager.Token(r.Context()),
		)

		ctx := context.WithValue(r.Context(), loggerContextKey, requestLogger)

		requestLogger.Info("request started",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		start := time.Now()

		lrw := newLoggingResponseWriter(w)
		next.ServeHTTP(lrw, r.WithContext(ctx))

		duration := time.Since(start)

		switch {
		case lrw.statusCode >= 500:
			requestLogger.Error("request completed", "status", lrw.statusCode, "bytes", lrw.bytes, "duration", duration.String())
		case lrw.statusCode >= 400:
			requestLogger.Warn("request completed", "status", lrw.statusCode, "bytes", lrw.bytes, "duration", duration.String())
		default:
			requestLogger.Info("request completed", "status", lrw.statusCode, "bytes", lrw.bytes, "duration", duration.String())
		}
	})
}
