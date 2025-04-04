package app

import "net/http"

type sessionKey string

const (
	SessionKeyUserId = sessionKey("userID")
	SessionKeyGuest  = sessionKey("guest")
)

func (s sessionKey) String() string {
	return string(s)
}

func (app *application) contextGetUserId(r *http.Request) int {
	userId, ok := r.Context().Value(SessionKeyUserId).(int)
	if !ok {
		panic("missing user id from context")
	}

	return userId
}
