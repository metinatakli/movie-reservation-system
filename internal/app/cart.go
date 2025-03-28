package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
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

	seatIds := input.SeatIdList
	showtimeSeats, err := app.seatRepo.GetSeatsByShowtimeAndSeatIds(r.Context(), showtimeID, seatIds)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if len(seatIds) != len(showtimeSeats.Seats) {
		app.badRequestResponse(w, r, fmt.Errorf("the provided seat IDs don't match the available seats for the showtime"))
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

func toApiCart(cart *Cart) api.Cart {
	return api.Cart{
		CartId:     cart.Id,
		ShowtimeId: cart.ShowtimeID,
		Seats:      toApiCartSeats(cart.Seats),
		HoldTime:   int(cartTTL.Seconds()),
		TotalPrice: cart.TotalPrice,
	}
}

func toApiCartSeats(cartSeats []CartSeat) []api.CartSeat {
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
	showtimeSeats *domain.ShowtimeSeats) (*Cart, error) {

	cart := createCartObj(showtimeID, showtimeSeats)
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

type Cart struct {
	Id         string `json:"-"`
	ShowtimeID int
	TotalPrice decimal.Decimal
	Seats      []CartSeat
}

type CartSeat struct {
	Id         int
	Row        int
	Col        int
	SeatType   string
	ExtraPrice decimal.Decimal
}

func createCartObj(showtimeID int, showtimeSeats *domain.ShowtimeSeats) Cart {
	id := uuid.New().String()
	seats := toCartSeats(showtimeSeats.Seats)
	totalPrice := calculateTotalPrice(showtimeSeats.Price, seats)

	return Cart{
		Id:         id,
		ShowtimeID: showtimeID,
		TotalPrice: totalPrice,
		Seats:      seats,
	}
}

func calculateTotalPrice(basePrice pgtype.Numeric, cartSeats []CartSeat) decimal.Decimal {
	baseFloat := toFloat64(basePrice)
	total := decimal.NewFromFloat(baseFloat)

	for _, v := range cartSeats {
		total = total.Add(v.ExtraPrice)
	}

	return total
}

func toCartSeats(seats []domain.Seat) []CartSeat {
	cartSeats := make([]CartSeat, len(seats))

	for i, seat := range seats {
		cartSeat := CartSeat{
			Id:       seat.ID,
			Row:      seat.Row,
			Col:      seat.Col,
			SeatType: seat.Type,
		}

		priceFloat := toFloat64(seat.ExtraPrice)
		cartSeat.ExtraPrice = decimal.NewFromFloat(priceFloat)

		cartSeats[i] = cartSeat
	}

	return cartSeats
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
