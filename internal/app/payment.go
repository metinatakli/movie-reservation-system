package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/redis/go-redis/v9"
)

func (app *application) CreateCheckoutSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionId := app.sessionManager.Token(r.Context())
	cartId, err := app.redis.Get(r.Context(), cartSessionKey(sessionId)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		app.serverErrorResponse(w, r, err)
		return
	}

	if cartId == "" {
		app.notFoundResponseWithErr(w, r, fmt.Errorf("there is no cart bound to the current session"))
		return
	}

	cartBytes, err := app.redis.Get(r.Context(), cartId).Bytes()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	var cart domain.Cart
	err = json.Unmarshal(cartBytes, &cart)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	cart.Id = cartId
	showtimeId := cart.ShowtimeID

	for _, seat := range cart.Seats {
		ownerSessionId, err := app.redis.Get(r.Context(), seatLockKey(showtimeId, seat.Id)).Result()
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		if sessionId != ownerSessionId {
			app.editConflictResponseWithErr(
				w,
				r,
				fmt.Errorf("seat %d doesn't belong to the current session", seat.Id),
			)
			return
		}
	}

	userId := app.contextGetUserId(r)
	user, err := app.userRepo.GetById(r.Context(), userId)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	checkoutSession, err := app.paymentProvider.CreateCheckoutSession(sessionId, user, cart)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	payment := domain.Payment{
		UserID:            userId,
		CheckoutSessionId: checkoutSession.ID,
		Amount:            cart.TotalPrice,
		Currency:          "USD",
		Status:            domain.PaymentStatusPending,
	}

	err = app.paymentRepo.Create(r.Context(), payment)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	resp := api.CheckoutSessionResponse{
		RedirectUrl: checkoutSession.URL,
	}

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
