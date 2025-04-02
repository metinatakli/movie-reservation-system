package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/metinatakli/movie-reservation-system/internal/mocks"
	"github.com/metinatakli/movie-reservation-system/internal/validator"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	testShowtimeID = 1
	testBasePrice  = 50.0
	maxSeats       = 8
)

var (
	testSeatIDs = []int{1, 2, 3}
	testSeats   = []domain.Seat{
		{ID: 1, Row: 1, Col: 1, Type: "Standard", ExtraPrice: pgtype.Numeric{Int: decimal.NewFromFloat(0).BigInt(), Valid: true}},
		{ID: 2, Row: 1, Col: 2, Type: "VIP", ExtraPrice: pgtype.Numeric{Int: decimal.NewFromFloat(15).BigInt(), Valid: true}},
		{ID: 3, Row: 1, Col: 3, Type: "Recliner", ExtraPrice: pgtype.Numeric{Int: decimal.NewFromFloat(10).BigInt(), Valid: true}},
	}
)

type MockSeatRepo struct {
	mock.Mock
	domain.SeatRepository
}

func (m *MockSeatRepo) GetSeatsByShowtimeAndSeatIds(ctx context.Context, showtimeID int, seatIDs []int) (*domain.ShowtimeSeats, error) {
	args := m.Called(ctx, showtimeID, seatIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ShowtimeSeats), args.Error(1)
}

type CartTestSuite struct {
	suite.Suite
	app           *application
	seatRepo      *MockSeatRepo
	redisClient   *mocks.MockRedisClient
	redisPipeline *mocks.MockTxPipeline
}

func (s *CartTestSuite) SetupTest() {
	s.seatRepo = new(MockSeatRepo)
	s.redisClient = new(mocks.MockRedisClient)
	s.redisPipeline = new(mocks.MockTxPipeline)

	s.app = newTestApplication(func(a *application) {
		a.seatRepo = s.seatRepo
		a.sessionManager = scs.New()
		a.redis = s.redisClient
	})
}

func TestCartSuite(t *testing.T) {
	suite.Run(t, new(CartTestSuite))
}

func (s *CartTestSuite) TestCreateCartHandler() {
	tests := []struct {
		name           string
		showtimeID     int
		input          api.CreateCartRequest
		setupMocks     func()
		wantStatus     int
		wantErrMessage string
		wantResponse   *api.CartResponse
	}{
		{
			name:           "should fail when showtime ID is zero or negative",
			showtimeID:     0,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "showtime ID must be greater than zero",
		},
		{
			name:       "should fail when seat list is empty",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: []int{},
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: fmt.Sprintf(validator.ErrMinLength, "1"),
		},
		{
			name:       "should fail when seat IDs contain negative numbers",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: []int{1, -2, 3},
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: validator.ErrDefaultInvalid,
		},
		{
			name:       "should fail when seat count exceeds maximum limit of 8",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: make([]int, maxSeats+1),
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: fmt.Sprintf(validator.ErrMaxLength, "8"),
		},
		{
			name:       "should fail when user already has an active cart",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: testSeatIDs,
			},
			setupMocks: func() {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringResult("existing-cart-id", nil))
			},
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "cannot create new cart if a cart already exists in session",
		},
		{
			name:       "should fail when database error occurs while fetching seats",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: testSeatIDs,
			},
			setupMocks: func() {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringCmd(context.Background(), ""))
				s.seatRepo.On("GetSeatsByShowtimeAndSeatIds", mock.Anything, 1, testSeatIDs).Return(nil, fmt.Errorf("database error"))
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name:       "should fail when requested seats are not available for showtime",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: testSeatIDs,
			},
			setupMocks: func() {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringCmd(context.Background(), ""))
				s.seatRepo.On("GetSeatsByShowtimeAndSeatIds", mock.Anything, 1, testSeatIDs).Return(&domain.ShowtimeSeats{
					Seats: testSeats[:1],
				}, nil)
			},
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "the provided seat IDs don't match the available seats for the showtime",
		},
		{
			name:       "should handle concurrent seat locking failures",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: testSeatIDs,
			},
			setupMocks: func() {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringCmd(context.Background(), ""))
				s.seatRepo.On("GetSeatsByShowtimeAndSeatIds", mock.Anything, 1, testSeatIDs).Return(&domain.ShowtimeSeats{
					Seats: testSeats,
				}, nil)
				s.redisClient.On("TxPipeline").Return(s.redisPipeline)
				s.redisPipeline.On("SetNX", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(redis.NewBoolCmd(context.Background(), false))
				s.redisPipeline.On("Exec", mock.Anything).Return([]redis.Cmder{
					redis.NewBoolResult(false, nil),
					redis.NewBoolResult(true, nil),
					redis.NewBoolResult(true, nil),
				}, nil)
			},
			wantStatus:     http.StatusConflict,
			wantErrMessage: ErrEditConflict,
		},
		{
			name:       "should handle Redis pipeline execution failures during cart creation",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: testSeatIDs,
			},
			setupMocks: func() {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringCmd(context.Background(), ""))
				s.seatRepo.On("GetSeatsByShowtimeAndSeatIds", mock.Anything, 1, testSeatIDs).Return(&domain.ShowtimeSeats{
					Seats: testSeats,
				}, nil)

				// First pipeline (tryLockSeats) should succeed
				s.redisClient.On("TxPipeline").Return(s.redisPipeline).Once()
				s.redisPipeline.On("SetNX", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(redis.NewBoolCmd(context.Background(), true))
				s.redisPipeline.On("Exec", mock.Anything).Return([]redis.Cmder{
					redis.NewBoolResult(true, nil),
					redis.NewBoolResult(true, nil),
					redis.NewBoolResult(true, nil),
				}, nil).Once()

				// Second pipeline (createCart) should fail
				s.redisClient.On("TxPipeline").Return(s.redisPipeline).Once()
				s.redisPipeline.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(redis.NewStatusCmd(context.Background(), "OK"))
				s.redisPipeline.On("SAdd", mock.Anything, mock.Anything, mock.Anything).Return(redis.NewIntCmd(context.Background(), 1))
				s.redisPipeline.On("Exec", mock.Anything).Return(nil, fmt.Errorf("redis pipeline execution failed")).Once()

				// Verify rollback behavior - ensure deletion methods are called at least once for each seat ID
				for _, seatID := range testSeatIDs {
					s.redisClient.On("Del", mock.Anything, []string{seatLockKey(1, seatID)}).Return(redis.NewIntCmd(context.Background(), 1)).Once()
				}

				for _, seatID := range testSeatIDs {
					s.redisClient.On("SRem", mock.Anything, seatSetKey(1), []interface{}{seatID}).Return(redis.NewIntCmd(context.Background(), 1)).Once()
				}
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name:       "should successfully create cart with valid input",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: testSeatIDs,
			},
			setupMocks: func() {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringCmd(context.Background(), ""))
				s.seatRepo.On("GetSeatsByShowtimeAndSeatIds", mock.Anything, 1, testSeatIDs).Return(&domain.ShowtimeSeats{
					Seats: testSeats,
					Price: pgtype.Numeric{Int: decimal.NewFromFloat(testBasePrice).BigInt(), Valid: true},
				}, nil)
				s.redisClient.On("TxPipeline").Return(s.redisPipeline)
				s.redisPipeline.On("SetNX", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(redis.NewBoolCmd(context.Background(), true))
				s.redisPipeline.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(redis.NewStatusCmd(context.Background(), "OK"))
				s.redisPipeline.On("SAdd", mock.Anything, mock.Anything, mock.Anything).Return(redis.NewIntCmd(context.Background(), 1))
				s.redisPipeline.On("Exec", mock.Anything).Return([]redis.Cmder{
					redis.NewBoolResult(true, nil),
					redis.NewBoolResult(true, nil),
					redis.NewBoolResult(true, nil),
				}, nil)
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.CartResponse{
				Cart: api.Cart{
					ShowtimeId: 1,
					Seats: []api.CartSeat{
						{Id: 1, Row: 1, Column: 1, Type: api.Standard, Price: decimal.NewFromFloat(0)},
						{Id: 2, Row: 1, Column: 2, Type: api.VIP, Price: decimal.NewFromFloat(15)},
						{Id: 3, Row: 1, Column: 3, Type: api.Recliner, Price: decimal.NewFromFloat(10)},
					},
					HoldTime:   int(cartTTL.Seconds()),
					TotalPrice: decimal.NewFromFloat(75),
				},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()

			defer s.seatRepo.AssertExpectations(s.T())
			defer s.redisClient.AssertExpectations(s.T())
			defer s.redisPipeline.AssertExpectations(s.T())

			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			w, r := executeRequest(s.T(), http.MethodPost, fmt.Sprintf("/showtimes/%d/cart", tt.showtimeID), tt.input)
			r = setupTestSession(s.T(), s.app, r, 1)

			handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s.app.CreateCartHandler(w, r, tt.showtimeID)
			}))
			handler = s.app.sessionManager.LoadAndSave(handler)
			handler.ServeHTTP(w, r)

			s.Equal(tt.wantStatus, w.Code)

			if tt.wantResponse != nil {
				var response api.CartResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				s.Require().NoError(err, "Failed to decode response")

				cmpOpts := cmpopts.IgnoreFields(api.Cart{}, "CartId")
				diff := cmp.Diff(tt.wantResponse, &response, cmpOpts)
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
