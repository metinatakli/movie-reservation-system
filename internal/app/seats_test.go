package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/metinatakli/movie-reservation-system/internal/mocks"
)

func TestGetSeatMapByShowtime(t *testing.T) {
	tests := []struct {
		name                   string
		showtimeID             int
		getSeatsByShowtimeFunc func(context.Context, int) (*domain.ShowtimeSeats, error)
		wantStatus             int
		wantResponse           *api.SeatMapResponse
		wantErrMessage         string
	}{
		{
			name:       "successful retrieval",
			showtimeID: 1,
			getSeatsByShowtimeFunc: func(ctx context.Context, id int) (*domain.ShowtimeSeats, error) {
				return &domain.ShowtimeSeats{
					TheaterID:   1,
					TheaterName: "Test Theater",
					HallID:      2,
					Seats: []domain.Seat{
						{ID: 1, Row: 1, Col: 1, Type: "Standard"},
						{ID: 2, Row: 1, Col: 2, Type: "Accessible"},
						{ID: 3, Row: 2, Col: 1, Type: "VIP"},
						{ID: 4, Row: 2, Col: 2, Type: "Recliner"},
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.SeatMapResponse{
				TheaterId:   1,
				TheaterName: "Test Theater",
				HallId:      2,
				ShowtimeId:  1,
				SeatRows: []api.SeatRow{
					{
						Row: 1,
						Seats: []api.Seat{
							{Id: 1, Row: 1, Column: 1, Type: api.Standard, Available: true},
							{Id: 2, Row: 1, Column: 2, Type: api.Accessible, Available: true},
						},
					},
					{
						Row: 2,
						Seats: []api.Seat{
							{Id: 3, Row: 2, Column: 1, Type: api.VIP, Available: true},
							{Id: 4, Row: 2, Column: 2, Type: api.Recliner, Available: true},
						},
					},
				},
			},
		},
		{
			name:           "invalid showtime ID",
			showtimeID:     0,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "showtime ID must be greater than zero",
		},
		{
			name:       "showtime not found",
			showtimeID: 999,
			getSeatsByShowtimeFunc: func(ctx context.Context, id int) (*domain.ShowtimeSeats, error) {
				return nil, domain.ErrRecordNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "server error",
			showtimeID: 1,
			getSeatsByShowtimeFunc: func(ctx context.Context, id int) (*domain.ShowtimeSeats, error) {
				return nil, fmt.Errorf("database error")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(func(a *application) {
				a.seatRepo = &mocks.MockSeatRepo{
					GetSeatsByShowtimeFunc: tt.getSeatsByShowtimeFunc,
				}
			})

			w, r := executeRequest(t, http.MethodGet, fmt.Sprintf("/showtimes/%d/seats", tt.showtimeID), nil)

			app.GetSeatMapByShowtime(w, r, tt.showtimeID)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("GetSeatMapByShowtime() status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantResponse != nil {
				var response api.SeatMapResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if diff := cmp.Diff(tt.wantResponse, &response); diff != "" {
					t.Errorf("GetSeatMapByShowtime() response mismatch (-want +got):\n%s", diff)
				}
			}

			checkErrorResponse(t, w, struct {
				wantStatus     int
				wantErrMessage string
			}{
				wantStatus:     tt.wantStatus,
				wantErrMessage: tt.wantErrMessage,
			})
		})
	}
}
