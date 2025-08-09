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
	"github.com/stripe/stripe-go/v82/refund"
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

	cartBytes, err := app.redis.Get(r.Context(), cartId).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			app.redis.Del(r.Context(), cartSessionKey(sessionId))
			app.notFoundResponseWithErr(w, r, fmt.Errorf("cart not found or has expired, please try again"))
			return
		}

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
			// If a seat lock is missing, the user's cart has expired
			if errors.Is(err, redis.Nil) {
				app.editConflictResponseWithErr(
					w,
					r,
					fmt.Errorf("your selections have expired, please select your seats again"),
				)

				return
			}

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

	checkoutSession, err := app.paymentProvider.CreateCheckoutSession(sessionId, user, cart, *payment)
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

// TODO: handle idempotency
func (app *Application) StripeWebhookHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		app.logger.Error("Error reading request body", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	event := stripe.Event{}

	if err := json.Unmarshal(payload, &event); err != nil {
		app.logger.Error("Webhook error while parsing basic request", "error", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	endpointSecret := app.config.Stripe.WebhookSecret
	signatureHeader := r.Header.Get("Stripe-Signature")
	event, err = webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
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
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		go app.handleCheckoutSessionCompleted(context.Background(), session)
	default:
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func (app *Application) handleCheckoutSessionCompleted(
	ctx context.Context,
	checkoutSession stripe.CheckoutSession) (err error) {

	defer func() {
		if err != nil {
			app.logger.Error("checkout session handling failed", "error", err, "checkoutSessionID", checkoutSession.ID)

			if checkoutSession.PaymentIntent == nil {
				app.logger.Error("payment intent is nil, cannot issue refund", "checkoutSessionID", checkoutSession.ID)
				return
			}

			refundParams := &stripe.RefundParams{
				PaymentIntent: stripe.String(checkoutSession.PaymentIntent.ID),
			}

			refundResp, refundErr := refund.New(refundParams)
			if refundErr != nil {
				app.logger.Error("failed to issue refund", "error", refundErr, "checkoutSessionID", checkoutSession.ID)
				return
			}

			app.logger.Info("refund issued", "refundID", refundResp.ID, "checkoutSessionID", checkoutSession.ID)

			dbErr := app.paymentRepo.UpdateStatus(ctx, checkoutSession.ID, domain.PaymentStatusRefunded, err.Error())
			if dbErr != nil {
				app.logger.Error("failed to save refund status to DB", "error", dbErr, "checkoutSessionID", checkoutSession.ID)
			}
		}
	}()

	// TODO: code for checking cart ownership logic is duplicated across some handlers, reduce the duplication
	cartId := checkoutSession.Metadata["cart_id"]
	cartBytes, err := app.redis.Get(ctx, cartId).Bytes()
	if err != nil {
		return err
	}

	var cart domain.Cart

	err = json.Unmarshal(cartBytes, &cart)
	if err != nil {
		return err
	}

	showtimeId := cart.ShowtimeID
	sessionId := checkoutSession.Metadata["session_id"]
	for _, seat := range cart.Seats {
		ownerSessionId, err := app.redis.Get(ctx, seatLockKey(showtimeId, seat.Id)).Result()
		if err != nil {
			return err
		}

		if ownerSessionId != sessionId {
			return err
		}
	}

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
		return fmt.Errorf("user_id is not in the expected format")
	}

	reservation := domain.Reservation{
		UserID:            userId,
		ShowtimeID:        showtimeId,
		CheckoutSessionID: checkoutSession.ID,
		ReservationSeats:  reservationSeats,
	}

	err = app.reservationRepo.Create(ctx, reservation)
	if err != nil {
		return err
	}

	// remove cart and seat locks
	// TODO: remove duplicated code
	pipe := app.redis.TxPipeline()

	for _, seat := range cart.Seats {
		pipe.Del(ctx, seatLockKey(showtimeId, seat.Id))
		pipe.SRem(ctx, seatSetKey(showtimeId), seat.Id)
	}

	pipe.Del(ctx, cartId)
	pipe.Del(ctx, cartSessionKey(sessionId))

	_, err = pipe.Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
