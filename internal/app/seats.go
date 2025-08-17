package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

// Redis Lua script to clean up expired seat locks and return currently valid locked seat IDs.
var filterValidLockSeats = redis.NewScript(`
	local setKey = KEYS[1]
	local showtimeId = ARGV[1]
	local cursor = "0"
	local batchSize = 100
	local expiredSeats = {}
	local validSeats = {}

	repeat
		local result = redis.call("SSCAN", setKey, cursor, "COUNT", batchSize)
		cursor = result[1]
		local seatIds = result[2]

		for _, seatId in ipairs(seatIds) do
			local lockKey = "seat_lock:" .. showtimeId .. ":" .. seatId
			if redis.call("EXISTS", lockKey) == 0 then
				table.insert(expiredSeats, seatId)
			else
				table.insert(validSeats, seatId)
			end
		end
	until cursor == "0"

	if #expiredSeats > 0 then
		redis.call("SREM", setKey, unpack(expiredSeats))
	end

	return validSeats
`)

func (app *Application) GetSeatMapByShowtime(
	w http.ResponseWriter,
	r *http.Request,
	showtimeID int) {

	logger := app.contextGetLogger(r)

	if showtimeID < 1 {
		app.badRequestResponse(w, r, fmt.Errorf("showtime ID must be greater than zero"))
		return
	}

	showtimeSeats, err := app.seatRepo.GetSeatsByShowtime(r.Context(), showtimeID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if len(showtimeSeats.Seats) == 0 {
		logger.Warn("seat map not found for showtime", "showtime_id", showtimeID)
		app.notFoundResponse(w, r)
		return
	}

	err = app.updateSeatAvailability(r.Context(), showtimeID, showtimeSeats)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	resp := toSeatMapResponse(showtimeID, showtimeSeats)

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) updateSeatAvailability(ctx context.Context, showtimeID int, showtimeSeats *domain.ShowtimeSeats) error {
	cmd := filterValidLockSeats.Run(ctx, app.redis, []string{seatSetKey(showtimeID)}, showtimeID)
	lockedSeatIds, err := cmd.Int64Slice()
	if err != nil {
		return fmt.Errorf("failed to run filterValidLockSeats script: %w", err)
	}

	reservedSeats, err := app.reservationRepo.GetSeatsByShowtimeId(ctx, showtimeID)
	if err != nil {
		return fmt.Errorf("failed to get reserved seats from DB: %w", err)
	}

	unavailableSeats := make(map[int]bool)

	for _, seatId := range lockedSeatIds {
		unavailableSeats[int(seatId)] = true
	}

	for _, reservationSeat := range reservedSeats {
		unavailableSeats[reservationSeat.SeatID] = true
	}

	for i := range showtimeSeats.Seats {
		if unavailableSeats[showtimeSeats.Seats[i].ID] {
			showtimeSeats.Seats[i].Available = false
		}
	}

	return nil
}

func toSeatMapResponse(showtimeID int, showtimeSeats *domain.ShowtimeSeats) api.SeatMapResponse {
	return api.SeatMapResponse{
		TheaterId:   showtimeSeats.TheaterID,
		TheaterName: showtimeSeats.TheaterName,
		HallId:      showtimeSeats.HallID,
		ShowtimeId:  showtimeID,
		SeatRows:    toSeatRows(showtimeSeats.Seats),
	}
}

func toSeatRows(seats []domain.Seat) []api.SeatRow {
	// Seats are pre-sorted by Row,Column (ascending).
	// This allows us to process them in a single pass without additional sorting or mapping.

	var seatRows []api.SeatRow
	currentRow := api.SeatRow{Row: seats[0].Row}

	for _, v := range seats {
		if v.Row != currentRow.Row {
			seatRows = append(seatRows, currentRow)
			currentRow = api.SeatRow{Row: v.Row}
		}

		currentRow.Seats = append(currentRow.Seats, api.Seat{
			Id:         v.ID,
			Row:        v.Row,
			Column:     v.Col,
			ExtraPrice: decimal.NewFromFloat(v.ExtraPrice),
			Type:       api.SeatType(v.Type),
			Available:  v.Available,
		})
	}

	seatRows = append(seatRows, currentRow)

	return seatRows
}
