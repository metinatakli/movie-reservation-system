package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/metinatakli/movie-reservation-system/internal/mocks"
	"github.com/metinatakli/movie-reservation-system/internal/validator"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ReservationsTestSuite struct {
	suite.Suite
	app             *application
	reservationRepo *mocks.MockReservationRepo
}

func (s *ReservationsTestSuite) SetupTest() {
	s.reservationRepo = new(mocks.MockReservationRepo)
	s.app = newTestApplication(func(a *application) {
		a.reservationRepo = s.reservationRepo
		a.sessionManager = scs.New()
	})
}

func TestReservationsSuite(t *testing.T) {
	suite.Run(t, new(ReservationsTestSuite))
}

func (s *ReservationsTestSuite) TestGetReservationsOfUserHandler() {
	tests := []struct {
		name           string
		setupSession   bool
		userId         int
		params         api.GetReservationsOfUserHandlerParams
		setupMock      func()
		wantStatus     int
		wantErrMessage string
		wantResponse   *api.UserReservationsResponse
	}{
		{
			name:         "invalid page number",
			setupSession: true,
			userId:       1,
			params: api.GetReservationsOfUserHandlerParams{
				Page:     ptr(0),
				PageSize: ptr(10),
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: fmt.Sprintf(validator.ErrMinValue, "1"),
		},
		{
			name:         "invalid page size",
			setupSession: true,
			userId:       1,
			params: api.GetReservationsOfUserHandlerParams{
				Page:     ptr(1),
				PageSize: ptr(0),
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: fmt.Sprintf(validator.ErrMinValue, "1"),
		},
		{
			name:           "no session",
			setupSession:   false,
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: ErrUnauthorizedAccess,
		},
		{
			name:         "database error",
			setupSession: true,
			userId:       1,
			params: api.GetReservationsOfUserHandlerParams{
				Page:     ptr(1),
				PageSize: ptr(10),
			},
			setupMock: func() {
				s.reservationRepo.On("GetReservationsSummariesByUserId", mock.Anything, 1, domain.Pagination{
					Page:     1,
					PageSize: 10,
				}).Return(nil, nil, fmt.Errorf("database error"))
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name:         "successful retrieval",
			setupSession: true,
			userId:       1,
			params: api.GetReservationsOfUserHandlerParams{
				Page:     ptr(1),
				PageSize: ptr(10),
			},
			setupMock: func() {
				s.reservationRepo.On("GetReservationsSummariesByUserId", mock.Anything, 1, domain.Pagination{
					Page:     1,
					PageSize: 10,
				}).Return(
					[]domain.ReservationSummary{
						{
							ReservationID:  1,
							MovieTitle:     "The Matrix",
							MoviePosterUrl: "https://example.com/matrix.jpg",
							ShowtimeDate:   time.Date(2024, 3, 15, 19, 0, 0, 0, time.UTC),
							TheaterName:    "Cinema City",
							HallName:       "Hall 1",
							CreatedAt:      time.Date(2024, 3, 10, 10, 0, 0, 0, time.UTC),
						},
					},
					&domain.Metadata{
						CurrentPage:  1,
						PageSize:     10,
						FirstPage:    1,
						LastPage:     1,
						TotalRecords: 1,
					},
					nil,
				)
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.UserReservationsResponse{
				Reservations: []api.ReservationSummary{
					{
						Id:             1,
						MovieTitle:     "The Matrix",
						MoviePosterUrl: "https://example.com/matrix.jpg",
						HallName:       "Hall 1",
						TheaterName:    "Cinema City",
						Date:           time.Date(2024, 3, 15, 19, 0, 0, 0, time.UTC),
						CreatedAt:      time.Date(2024, 3, 10, 10, 0, 0, 0, time.UTC),
					},
				},
				Metadata: api.Metadata{
					CurrentPage:  1,
					PageSize:     10,
					FirstPage:    1,
					LastPage:     1,
					TotalRecords: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()

			defer s.reservationRepo.AssertExpectations(s.T())

			if tt.setupMock != nil {
				tt.setupMock()
			}

			w, r := executeRequest(s.T(), http.MethodGet, "/users/me/reservations", nil)

			if tt.setupSession {
				r = setupTestSession(s.T(), s.app, r, tt.userId)
			}

			q := r.URL.Query()
			if tt.params.Page != nil {
				q.Add("page", fmt.Sprintf("%d", *tt.params.Page))
			}
			if tt.params.PageSize != nil {
				q.Add("pageSize", fmt.Sprintf("%d", *tt.params.PageSize))
			}
			r.URL.RawQuery = q.Encode()

			handler := s.app.requireAuthentication(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s.app.GetReservationsOfUserHandler(w, r, tt.params)
			}))
			handler = s.app.sessionManager.LoadAndSave(handler)
			handler.ServeHTTP(w, r)

			s.Equal(tt.wantStatus, w.Code)

			if tt.wantResponse != nil {
				var response api.UserReservationsResponse
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

func (s *ReservationsTestSuite) TestGetUserReservationById() {
	tests := []struct {
		name           string
		setupSession   bool
		userId         int
		reservationId  int
		setupMock      func()
		wantStatus     int
		wantErrMessage string
		wantResponse   *api.ReservationDetailResponse
	}{
		{
			name:           "invalid reservation id",
			setupSession:   true,
			userId:         1,
			reservationId:  0,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "reservation id must be greater than zero",
		},
		{
			name:           "no session",
			setupSession:   false,
			reservationId:  1,
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: ErrUnauthorizedAccess,
		},
		{
			name:          "reservation not found",
			setupSession:  true,
			userId:        1,
			reservationId: 1,
			setupMock: func() {
				s.reservationRepo.On("GetByReservationIdAndUserId", mock.Anything, 1, 1).
					Return(nil, domain.ErrRecordNotFound)
			},
			wantStatus:     http.StatusNotFound,
			wantErrMessage: ErrNotFound,
		},
		{
			name:          "database error",
			setupSession:  true,
			userId:        1,
			reservationId: 1,
			setupMock: func() {
				s.reservationRepo.On("GetByReservationIdAndUserId", mock.Anything, 1, 1).
					Return(nil, fmt.Errorf("database error"))
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name:          "successful retrieval",
			setupSession:  true,
			userId:        1,
			reservationId: 1,
			setupMock: func() {
				s.reservationRepo.On("GetByReservationIdAndUserId", mock.Anything, 1, 1).
					Return(&domain.ReservationDetail{
						ReservationSummary: domain.ReservationSummary{
							ReservationID:  1,
							MovieTitle:     "The Matrix",
							MoviePosterUrl: "https://example.com/matrix.jpg",
							ShowtimeDate:   time.Date(2024, 3, 15, 19, 0, 0, 0, time.UTC),
							TheaterName:    "Cinema City",
							HallName:       "Hall 1",
							CreatedAt:      time.Date(2024, 3, 10, 10, 0, 0, 0, time.UTC),
						},
						Seats: []domain.ReservationDetailSeat{
							{Row: "1", Col: 1, Type: "standard"},
							{Row: "1", Col: 2, Type: "vip"},
						},
						TheaterAmenities: []domain.Amenity{
							{ID: 1, Name: "Parking", Description: "Free parking available"},
						},
						HallAmenities: []domain.Amenity{
							{ID: 2, Name: "3D Glasses", Description: "3D glasses provided"},
						},
						TotalPrice: decimal.NewFromFloat(25.50),
					}, nil)
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.ReservationDetailResponse{
				Id:             1,
				MovieTitle:     "The Matrix",
				MoviePosterUrl: "https://example.com/matrix.jpg",
				Date:           time.Date(2024, 3, 15, 19, 0, 0, 0, time.UTC),
				TheaterName:    "Cinema City",
				HallName:       "Hall 1",
				CreatedAt:      time.Date(2024, 3, 10, 10, 0, 0, 0, time.UTC),
				TotalPrice:     decimal.NewFromFloat(25.50),
				Seats: []api.ReservationSeat{
					{Row: "1", Column: 1, Type: "standard"},
					{Row: "1", Column: 2, Type: "vip"},
				},
				TheaterAmenities: &[]api.Amenity{
					{Id: 1, Name: "Parking", Description: "Free parking available"},
				},
				HallAmenities: &[]api.Amenity{
					{Id: 2, Name: "3D Glasses", Description: "3D glasses provided"},
				},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()

			defer s.reservationRepo.AssertExpectations(s.T())

			if tt.setupMock != nil {
				tt.setupMock()
			}

			w, r := executeRequest(s.T(), http.MethodGet, fmt.Sprintf("/reservations/%d", tt.reservationId), nil)

			if tt.setupSession {
				r = setupTestSession(s.T(), s.app, r, tt.userId)
			}

			handler := s.app.requireAuthentication(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s.app.GetUserReservationById(w, r, tt.reservationId)
			}))
			handler = s.app.sessionManager.LoadAndSave(handler)
			handler.ServeHTTP(w, r)

			s.Equal(tt.wantStatus, w.Code)

			if tt.wantResponse != nil {
				var response api.ReservationDetailResponse
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
