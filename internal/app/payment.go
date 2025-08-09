package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

const (
	maxBodyBytes = int64(65536)
)

func (app *Application) CreateCheckoutSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionId := app.sessionManager.Token(r.Context())
	cartId, err := app.redis.Get(r.Context(), cartSessionKey(sessionId)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			app.notFoundResponseWithErr(w, r, fmt.Errorf("there is no cart bound to the current session"))
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	cart, err := app.getAndVerifyCart(r.Context(), cartId, sessionId)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCartNotFound):
			app.notFoundResponseWithErr(w, r, err)
		case errors.Is(err, domain.ErrSeatLockExpired), errors.Is(err, domain.ErrSeatConflict):
			app.editConflictResponseWithErr(w, r, err)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	userId := app.contextGetUserId(r)
	user, err := app.userRepo.GetById(r.Context(), userId)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	payment := &domain.Payment{
		UserID:   userId,
		Amount:   cart.TotalPrice,
		Currency: "USD",
		Status:   domain.PaymentStatusPending,
	}

	err = app.paymentRepo.Create(r.Context(), payment)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	checkoutSession, err := app.paymentProvider.CreateCheckoutSession(sessionId, user, *cart, *payment)
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

func (app *Application) StripeWebhookHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		app.logger.Error("Error reading request body", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	endpointSecret := app.config.Stripe.WebhookSecret
	signatureHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
	if err != nil {
		app.logger.Error("Webhook signature verification failed", "error", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession

		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			app.logger.Error("error parsing webhook JSON", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		app.handleCheckoutSessionCompleted(w, r, session)
	default:
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
		w.WriteHeader(http.StatusOK)
	}
}

func (app *Application) handleCheckoutSessionCompleted(
	w http.ResponseWriter,
	r *http.Request,
	checkoutSession stripe.CheckoutSession) {

	paymentIdStr := checkoutSession.Metadata["payment_id"]
	if paymentIdStr == "" {
		app.badRequestResponse(w, r, fmt.Errorf("payment_id is missing in the checkout session metadata"))
		return
	}

	paymentId, err := strconv.Atoi(paymentIdStr)
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("payment_id is not in the expected format: %w", err))
		return
	}

	payment, err := app.paymentRepo.GetById(r.Context(), paymentId)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRecordNotFound):
			app.notFoundResponseWithErr(w, r, fmt.Errorf("payment not found: %w", err))
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("failed to get payment by id: %w", err))
		}

		return
	}

	if payment.Status == domain.PaymentStatusCompleted {
		app.logger.Info("idempotent request: payment already completed", "payment_id", paymentId)
		w.WriteHeader(http.StatusOK)
		return
	}

	if payment.Status != domain.PaymentStatusPending {
		app.editConflictResponseWithErr(w, r, fmt.Errorf("payment status is not pending: %s", payment.Status))
		return
	}

	cartId := checkoutSession.Metadata["cart_id"]
	sessionId := checkoutSession.Metadata["session_id"]

	cart, err := app.getAndVerifyCart(r.Context(), cartId, sessionId)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCartNotFound):
			app.notFoundResponseWithErr(w, r, err)
		case errors.Is(err, domain.ErrSeatLockExpired), errors.Is(err, domain.ErrSeatConflict):
			app.editConflictResponseWithErr(w, r, err)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	showtimeId := cart.ShowtimeID

	reservationSeats := make([]domain.ReservationSeat, len(cart.Seats))
	for i, seat := range cart.Seats {
		reservationSeat := domain.ReservationSeat{
			ShowtimeID: showtimeId,
			SeatID:     seat.Id,
		}

		reservationSeats[i] = reservationSeat
	}

	userId, err := strconv.Atoi(checkoutSession.Metadata["user_id"])
	if err != nil || userId == 0 {
		app.badRequestResponse(w, r, fmt.Errorf("user_id is missing or not in the expected format: %w", err))
		return
	}

	reservation := domain.Reservation{
		UserID:            userId,
		ShowtimeID:        showtimeId,
		CheckoutSessionID: checkoutSession.ID,
		PaymentID:         paymentId,
		ReservationSeats:  reservationSeats,
	}

	err = app.reservationRepo.Create(r.Context(), reservation)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("failed to create reservation: %w", err))
		return
	}

	// remove cart and seat locks
	// TODO: remove duplicated code
	pipe := app.redis.TxPipeline()

	for _, seat := range cart.Seats {
		pipe.Del(r.Context(), seatLockKey(showtimeId, seat.Id))
		pipe.SRem(r.Context(), seatSetKey(showtimeId), seat.Id)
	}

	pipe.Del(r.Context(), cartId)
	pipe.Del(r.Context(), cartSessionKey(sessionId))

	_, err = pipe.Exec(r.Context())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (app *Application) getAndVerifyCart(ctx context.Context, cartId, sessionId string) (*domain.Cart, error) {
	cartBytes, err := app.redis.Get(ctx, cartId).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			app.redis.Del(ctx, cartSessionKey(sessionId))
			return nil, domain.ErrCartNotFound
		}

		return nil, err
	}

	var cart domain.Cart
	if err := json.Unmarshal(cartBytes, &cart); err != nil {
		return nil, err
	}

	cart.Id = cartId
	showtimeId := cart.ShowtimeID

	for _, seat := range cart.Seats {
		ownerSessionId, err := app.redis.Get(ctx, seatLockKey(showtimeId, seat.Id)).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return nil, domain.ErrSeatLockExpired
			}
			return nil, err
		}

		if sessionId != ownerSessionId {
			return nil, domain.ErrSeatConflict
		}
	}

	return &cart, nil
}
