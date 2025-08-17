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

var lockSeatsScript = redis.NewScript(`
    -- KEYS = seat lock keys (e.g., seat_lock:123:1, seat_lock:123:2 etc.)
    -- ARGV = [sessionID, ttl]

    for i=1, #KEYS do
        if redis.call("EXISTS", KEYS[i]) == 1 then
            return {err = "seat already locked"} -- Return an error indicator
        end
    end

    for i=1, #KEYS do
        redis.call("SET", KEYS[i], ARGV[1], "EX", ARGV[2])
    end

    return "OK"
`)

func (app *Application) CreateCartHandler(w http.ResponseWriter, r *http.Request, showtimeID int) {
	logger := app.contextGetLogger(r)

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
		logger.Error("failed to check for existing cart in redis", "error", err)
		app.serverErrorResponse(w, r, err)
		return
	}

	if cartId != "" {
		logger.Warn("cart creation attempt rejected: a cart already exists for this session")
		app.badRequestResponse(w, r, fmt.Errorf("cannot create new cart if a cart already exists in session"))
		return
	}

	// TODO: Reserved seats can be moved to Redis as well until showtime start time is passed.
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
			logger.Warn("cart creation conflict: user selected an already reserved seat", "seat_id", seatID)
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
		logger.Warn("cart creation failed: one or more requested seat IDs do not exist for the showtime", "requested_seats", seatIds)
		app.notFoundResponse(w, r)
		return
	}

	err = app.tryLockSeats(r.Context(), seatIds, showtimeID, sessionID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSeatAlreadyReserved):
			logger.Warn("cart creation conflict due to race condition: user selected an already locked seat")
			app.editConflictResponseWithErr(w, r, fmt.Errorf("some of the selected seats are already reserved"))
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("seats couldn't be acquired: %w", err))
		}

		return
	}

	cart, err := app.createCart(r.Context(), seatIds, showtimeID, sessionID, showtimeSeats)
	if err != nil {
		logger.Error("cart creation process failed", "error", err)
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

func (app *Application) tryLockSeats(ctx context.Context, seatIDs []int, showtimeID int, sessionID string) error {
	keys := make([]string, len(seatIDs))
	for i, seatID := range seatIDs {
		keys[i] = seatLockKey(showtimeID, seatID)
	}

	err := lockSeatsScript.Run(ctx, app.redis, keys, sessionID, int(seatLockTTL.Seconds())).Err()
	if err != nil {
		if redis.HasErrorPrefix(err, "seat already locked") {
			return domain.ErrSeatAlreadyReserved
		}

		return err
	}

	return nil
}

func (app *Application) createCart(
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

	seatIdInterfaces := make([]interface{}, len(seatIDs))
	for i, seatID := range seatIDs {
		seatIdInterfaces[i] = seatID
	}
	cartPipe.SAdd(ctx, seatSetKey(showtimeID), seatIdInterfaces...)

	cartPipe.Set(ctx, cartSessionKey(sessionID), cart.Id, cartTTL)
	cartPipe.Set(ctx, cart.Id, cartBytes, cartTTL)

	_, err = cartPipe.Exec(ctx)
	if err != nil {
		app.rollbackSeatLocks(ctx, showtimeID, seatIDs)
		return nil, err
	}

	return &cart, nil
}

func (app *Application) rollbackSeatLocks(ctx context.Context, showtimeID int, seatIDs []int) {
	lockKeys := make([]string, len(seatIDs))
	seatIDInterfaces := make([]interface{}, len(seatIDs))

	for i, seatID := range seatIDs {
		lockKeys[i] = seatLockKey(showtimeID, seatID)
		seatIDInterfaces[i] = seatID
	}

	pipe := app.redis.TxPipeline()
	pipe.Del(ctx, lockKeys...)
	pipe.SRem(ctx, seatSetKey(showtimeID), seatIDInterfaces...)

	_, err := pipe.Exec(ctx)
	if err != nil {
		app.logger.Error("failed to rollback seat locks", "error", err)
		return
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

func (app *Application) DeleteCartHandler(w http.ResponseWriter, r *http.Request, showtimeID int) {
	logger := app.contextGetLogger(r)

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
		if errors.Is(err, redis.Nil) {
			// The session points to a cart that no longer exists, delete the session key
			logger.Warn("dangling cart session key found and cleaned up", "dangling_cart_id", cartId)
			app.redis.Del(r.Context(), cartSessionKey(sessionID))
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	var cart domain.Cart

	err = json.Unmarshal(cartBytes, &cart)
	if err != nil {
		logger.Error("failed to unmarshal cart from redis", "cart_id", cartId, "error", err)
		app.serverErrorResponse(w, r, err)
		return
	}

	if cart.ShowtimeID != showtimeID {
		logger.Warn(
			"cart deletion attempt with mismatched showtime ID in URL",
			"cart_showtime_id", cart.ShowtimeID,
			"url_showtime_id", showtimeID,
		)
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

func (app *Application) migrateSessionData(ctx context.Context, oldSessionId, newSessionId string) error {
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
