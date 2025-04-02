package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/metinatakli/movie-reservation-system/internal/mocks"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type SeatsTestSuite struct {
	suite.Suite
	app         *application
	seatRepo    *mocks.MockSeatRepo
	redisClient *mocks.MockRedisClient
}

func (s *SeatsTestSuite) SetupTest() {
	s.seatRepo = new(mocks.MockSeatRepo)
	s.redisClient = new(mocks.MockRedisClient)

	s.app = newTestApplication(func(a *application) {
		a.seatRepo = s.seatRepo
		a.redis = s.redisClient
	})
}

func TestSeatsSuite(t *testing.T) {
	suite.Run(t, new(SeatsTestSuite))
}

func (s *SeatsTestSuite) TestGetSeatMapByShowtime() {
	tests := []struct {
		name           string
		showtimeID     int
		setupMocks     func()
		wantStatus     int
		wantResponse   *api.SeatMapResponse
		wantErrMessage string
	}{
		{
			name:           "should fail when showtime ID is zero or negative",
			showtimeID:     0,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "showtime ID must be greater than zero",
		},
		{
			name:       "should fail when seat data related to showtime is not found",
			showtimeID: 999,
			setupMocks: func() {
				s.seatRepo.On("GetSeatsByShowtime", mock.Anything, 999).Return(&domain.ShowtimeSeats{}, nil)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "should fail when database error occurs while fetching seats",
			showtimeID: 1,
			setupMocks: func() {
				s.seatRepo.On("GetSeatsByShowtime", mock.Anything, 1).Return(nil, fmt.Errorf("database error"))
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name:       "should fail when redis script execution fails",
			showtimeID: 1,
			setupMocks: func() {
				s.seatRepo.On("GetSeatsByShowtime", mock.Anything, 1).Return(&domain.ShowtimeSeats{
					TheaterID:   1,
					TheaterName: "Test Theater",
					HallID:      2,
					Seats: []domain.Seat{
						{ID: 1, Row: 1, Col: 1, Type: "Standard", Available: true},
						{ID: 2, Row: 1, Col: 2, Type: "Accessible", Available: true},
					},
				}, nil)

				s.redisClient.On("EvalSha", mock.Anything, mock.Anything, []string{seatSetKey(1)}, mock.Anything).
					Return(redis.NewCmdResult(nil, fmt.Errorf("redis error")))
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name:       "should return seat map with valid input",
			showtimeID: 1,
			setupMocks: func() {
				s.seatRepo.On("GetSeatsByShowtime", mock.Anything, 1).Return(&domain.ShowtimeSeats{
					TheaterID:   1,
					TheaterName: "Test Theater",
					HallID:      2,
					Seats: []domain.Seat{
						{ID: 1, Row: 1, Col: 1, Type: "Standard", Available: true},
						{ID: 2, Row: 1, Col: 2, Type: "Accessible", Available: true},
						{ID: 3, Row: 2, Col: 1, Type: "VIP", Available: true},
						{ID: 4, Row: 2, Col: 2, Type: "Recliner", Available: true},
					},
				}, nil)

				s.redisClient.On("EvalSha", mock.Anything, mock.Anything, []string{seatSetKey(1)}, mock.Anything).
					Return(redis.NewCmdResult([]interface{}{"2", "4"}, nil))
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
							{Id: 2, Row: 1, Column: 2, Type: api.Accessible, Available: false},
						},
					},
					{
						Row: 2,
						Seats: []api.Seat{
							{Id: 3, Row: 2, Column: 1, Type: api.VIP, Available: true},
							{Id: 4, Row: 2, Column: 2, Type: api.Recliner, Available: false},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()

			defer s.seatRepo.AssertExpectations(s.T())
			defer s.redisClient.AssertExpectations(s.T())

			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			w, r := executeRequest(s.T(), http.MethodGet, fmt.Sprintf("/showtimes/%d/seats", tt.showtimeID), nil)
			s.app.GetSeatMapByShowtime(w, r, tt.showtimeID)

			s.Equal(tt.wantStatus, w.Code)

			if tt.wantResponse != nil {
				var response api.SeatMapResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				s.Require().NoError(err, "Failed to decode response")

				diff := cmp.Diff(tt.wantResponse, &response)
				s.Empty(diff, "Response mismatch (-want +got):\n%s", diff)
			}

			checkErrorResponse(s.T(), w, struct {
				wantStatus     int
				wantErrMessage string
			}{
				wantStatus:     tt.wantStatus,
				wantErrMessage: tt.wantErrMessage,
			})
		})
	}
}
