package app

import (
	"fmt"
	"net/http"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

func (app *application) GetSeatMapByShowtime(
	w http.ResponseWriter,
	r *http.Request,
	showtimeID int) {

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
		app.notFoundResponse(w, r)
		return
	}

	resp := toSeatMapResponse(showtimeID, showtimeSeats)

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
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

		apiSeat := api.Seat{
			Id:        v.ID,
			Row:       v.Row,
			Column:    v.Col,
			Type:      api.SeatType(v.Type),
			Available: true, // TODO: After adding reservations, change this
		}

		currentRow.Seats = append(currentRow.Seats, apiSeat)
	}

	seatRows = append(seatRows, currentRow)

	return seatRows
}
