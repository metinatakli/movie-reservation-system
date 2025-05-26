package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	seatLockTTL = 10 * time.Minute
	cartTTL     = 10 * time.Minute
)

func (app *application) CreateCartHandler(w http.ResponseWriter, r *http.Request, showtimeID int) {
	if showtimeID < 1 {
		app.badRequestResponse(w, r, fmt.Errorf("showtime ID must be greater than zero"))
		return
	}

	var input api.CreateCartRequest

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	err = app.validator.Struct(input)
	if err != nil {
		app.failedValidationResponse(w, r, err)
		return
	}

	sessionID := app.sessionManager.Token(r.Context())
	cartId, err := app.redis.Get(r.Context(), cartSessionKey(sessionID)).Result()
	if err != nil && err != redis.Nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if cartId != "" {
		app.badRequestResponse(w, r, fmt.Errorf("cannot create new cart if a cart already exists in session"))
		return
	}

	reservedSeats, err := app.reservationRepo.GetSeatsByShowtimeId(r.Context(), showtimeID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	reservedSeatIds := make(map[int]bool, len(reservedSeats))
	for _, rs := range reservedSeats {
		reservedSeatIds[rs.SeatID] = true
	}

	seatIds := input.SeatIdList

	for _, seatID := range seatIds {
		if reservedSeatIds[seatID] {
			app.editConflictResponseWithErr(w, r, fmt.Errorf("some of the selected seats are already reserved"))
			return
		}
	}

	showtimeSeats, err := app.seatRepo.GetSeatsByShowtimeAndSeatIds(r.Context(), showtimeID, seatIds)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if len(seatIds) != len(showtimeSeats.Seats) {
		app.notFoundResponse(w, r)
		return
	}

	err = app.tryLockSeats(r.Context(), seatIds, showtimeID, sessionID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSeatAlreadyReserved):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("seats couldn't be acquired: %w", err))
		}

		return
	}

	cart, err := app.createCart(r.Context(), seatIds, showtimeID, sessionID, showtimeSeats)
	if err != nil {
		app.serverErrorResponse(w, r, fmt.Errorf("cart couldn't be created: %w", err))
		return
	}

	resp := api.CartResponse{
		Cart: toApiCart(cart),
	}

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func toApiCart(cart *domain.Cart) api.Cart {
	return api.Cart{
		CartId:       cart.Id,
		ShowtimeId:   cart.ShowtimeID,
		MovieName:    cart.MovieName,
		TheaterName:  cart.TheaterName,
		HallName:     cart.HallName,
		ShowtimeDate: cart.Date.Format(time.RFC1123),
		Seats:        toApiCartSeats(cart.Seats),
		HoldTime:     int(cartTTL.Seconds()),
		BasePrice:    cart.BasePrice,
		TotalPrice:   cart.TotalPrice,
	}
}

func toApiCartSeats(cartSeats []domain.CartSeat) []api.CartSeat {
	apiCartSeats := make([]api.CartSeat, len(cartSeats))

	for i, v := range cartSeats {
		apiCartSeat := api.CartSeat{
			Id:     v.Id,
			Row:    v.Row,
			Column: v.Col,
			Type:   api.SeatType(v.SeatType),
			Price:  v.ExtraPrice,
		}

		apiCartSeats[i] = apiCartSeat
	}

	return apiCartSeats
}

func (app *application) tryLockSeats(ctx context.Context, seatIDs []int, showtimeID int, sessionID string) error {
	lockPipe := app.redis.TxPipeline()

	for _, seatID := range seatIDs {
		lockPipe.SetNX(ctx, seatLockKey(showtimeID, seatID), sessionID, seatLockTTL)
	}

	lockCmds, err := lockPipe.Exec(ctx)
	if err != nil {
		app.logger.Debug("Pipeline for acquiring seat locks failed")
		return err
	}

	for i, seatID := range seatIDs {
		lockKey := seatLockKey(showtimeID, seatID)
		acquired, err := lockCmds[i].(*redis.BoolCmd).Result()

		if err != nil || !acquired {
			app.logger.Debug(fmt.Sprintf("Seat lock %s could not be acquired", lockKey))

			for j := 0; j < i; j++ {
				prevLockKey := seatLockKey(showtimeID, seatIDs[j])

				_, delErr := app.redis.Del(ctx, prevLockKey).Result()
				if delErr != nil {
					app.logger.Debug(fmt.Sprintf("Failed to delete seat lock %s: %v", prevLockKey, delErr))
				}
			}

			if err != nil {
				return err
			}

			return domain.ErrSeatAlreadyReserved
		}
	}

	return nil
}

func (app *application) createCart(
	ctx context.Context,
	seatIDs []int,
	showtimeID int,
	sessionID string,
	showtimeSeats *domain.ShowtimeSeats) (*domain.Cart, error) {

	cart := domain.NewCart(showtimeID, showtimeSeats)
	cartBytes, err := json.Marshal(cart)
	if err != nil {
		app.rollbackSeatLocks(ctx, showtimeID, seatIDs)
		return nil, err
	}

	cartPipe := app.redis.TxPipeline()
	seatSetKey := seatSetKey(showtimeID)

	for _, seatID := range seatIDs {
		cartPipe.SAdd(ctx, seatSetKey, seatID)
	}

	cartPipe.Set(ctx, cartSessionKey(sessionID), cart.Id, cartTTL)
	cartPipe.Set(ctx, cart.Id, cartBytes, cartTTL)

	_, err = cartPipe.Exec(ctx)
	if err != nil {
		app.rollbackSeatLocks(ctx, showtimeID, seatIDs)
		return nil, err
	}

	return &cart, nil
}

func (app *application) rollbackSeatLocks(ctx context.Context, showtimeID int, seatIDs []int) {
	seatSetKey := seatSetKey(showtimeID)

	for _, seatID := range seatIDs {
		lockKey := seatLockKey(showtimeID, seatID)

		if _, err := app.redis.Del(ctx, lockKey).Result(); err != nil {
			app.logger.Debug(fmt.Sprintf("Failed to delete seat lock %s: %v", lockKey, err))
		}

		if _, err := app.redis.SRem(ctx, seatSetKey, seatID).Result(); err != nil {
			app.logger.Debug(fmt.Sprintf("Failed to remove seat %d from set %s: %v", seatID, seatSetKey, err))
		}
	}
}

func cartSessionKey(sessionID string) string {
	return fmt.Sprintf("cart:%s", sessionID)
}

func seatLockKey(showtimeID, seatID int) string {
	return fmt.Sprintf("seat_lock:%d:%d", showtimeID, seatID)
}

func seatSetKey(showtimeID int) string {
	return fmt.Sprintf("seat_locks:%d", showtimeID)
}

func (app *application) DeleteCartHandler(w http.ResponseWriter, r *http.Request, showtimeID int) {
	if showtimeID < 1 {
		app.badRequestResponse(w, r, fmt.Errorf("showtime ID must be greater than zero"))
		return
	}

	sessionID := app.sessionManager.Token(r.Context())

	cartId, err := app.redis.Get(r.Context(), cartSessionKey(sessionID)).Result()
	if err != nil && err != redis.Nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if cartId == "" {
		app.notFoundResponse(w, r)
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

	if cart.ShowtimeID != showtimeID {
		app.notFoundResponse(w, r)
		return
	}

	pipe := app.redis.TxPipeline()

	for _, seat := range cart.Seats {
		pipe.Del(r.Context(), seatLockKey(showtimeID, seat.Id))
		pipe.SRem(r.Context(), seatSetKey(showtimeID), seat.Id)
	}

	pipe.Del(r.Context(), cartId)
	pipe.Del(r.Context(), cartSessionKey(sessionID))

	_, err = pipe.Exec(r.Context())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *application) migrateSessionData(ctx context.Context, oldSessionId, newSessionId string) error {
	cartId, err := app.redis.Get(ctx, cartSessionKey(oldSessionId)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("failed to get cart ID for session %s: %w", oldSessionId, err)
	}

	if cartId == "" {
		return nil
	}

	var cart domain.Cart
	cartBytes, err := app.redis.Get(ctx, cartId).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return fmt.Errorf("failed to get cart data for session %s: %w", oldSessionId, err)
	}

	err = json.Unmarshal(cartBytes, &cart)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cart data for session %s: %w", oldSessionId, err)
	}

	ttl, err := app.redis.TTL(ctx, cartId).Result()
	if err != nil {
		return fmt.Errorf("failed to get TTL for cart ID %s: %w", cartId, err)
	}

	if ttl <= 0 {
		// Key either doesn't exist (-2) or is persistent (-1), put for safety
		return nil
	}

	newTTL := ttl + 3*time.Minute
	showtimeId := cart.ShowtimeID
	lockKeys := make([]string, len(cart.Seats))

	for i, seat := range cart.Seats {
		lockKeys[i] = seatLockKey(showtimeId, seat.Id)
	}

	err = app.redis.Watch(ctx, func(tx *redis.Tx) error {
		for _, lockKey := range lockKeys {
			sessionId, err := tx.Get(ctx, lockKey).Result()
			if err != nil && !errors.Is(err, redis.Nil) {
				return err
			}

			if sessionId != oldSessionId {
				return fmt.Errorf("seat doesn't belong to current session")
			}
		}

		pipe := tx.TxPipeline()

		for _, lockKey := range lockKeys {
			pipe.Set(ctx, lockKey, newSessionId, newTTL).Result()
		}

		_, err := pipe.Exec(ctx)

		return err
	}, lockKeys...)

	if err != nil {
		return fmt.Errorf(
			"failed to migrate seat locks from old session %s to new session %s: %w",
			oldSessionId,
			newSessionId,
			err)
	}

	pipe := app.redis.TxPipeline()

	pipe.Expire(ctx, cartId, newTTL)
	pipe.Set(ctx, cartSessionKey(newSessionId), cartId, newTTL)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute Redis pipeline for session migration: %w", err)
	}

	return nil
}
