package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

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
)

// TODO: Add testify to mock easily
// TODO: Add test case for failures during cart creation. Ensure rollback behavior is correctly executed
func TestCreateCartHandler(t *testing.T) {
	tests := []struct {
		name           string
		showtimeID     int
		input          api.CreateCartRequest
		getSeatsFunc   func(context.Context, int, []int) (*domain.ShowtimeSeats, error)
		redisGetFunc   func(context.Context, string) *redis.StringCmd
		redisSetFunc   func(context.Context, string, interface{}, time.Duration) *redis.StatusCmd
		redisSAddFunc  func(context.Context, string, ...interface{}) *redis.IntCmd
		redisSetNXFunc func(context.Context, string, interface{}, time.Duration) *redis.BoolCmd
		redisDelFunc   func(context.Context, ...string) *redis.IntCmd
		redisSRemFunc  func(context.Context, string, ...interface{}) *redis.IntCmd
		redisExecFunc  func(context.Context, []int) ([]redis.Cmder, error)
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
				SeatIdList: []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
			},
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: fmt.Sprintf(validator.ErrMaxLength, "8"),
		},
		{
			name:       "should fail when user already has an active cart",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: []int{1, 2, 3},
			},
			redisGetFunc: func(ctx context.Context, key string) *redis.StringCmd {
				return redis.NewStringResult("existing-cart-id", nil)
			},
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "cannot create new cart if a cart already exists in session",
		},
		{
			name:       "should fail when database error occurs while fetching seats",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: []int{1, 2, 3},
			},
			redisGetFunc: func(ctx context.Context, key string) *redis.StringCmd {
				return redis.NewStringCmd(ctx, "")
			},
			getSeatsFunc: func(ctx context.Context, showtimeID int, seatIDs []int) (*domain.ShowtimeSeats, error) {
				return nil, fmt.Errorf("database error")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name:       "should fail when requested seats are not available for showtime",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: []int{1, 2, 3},
			},
			redisGetFunc: func(ctx context.Context, key string) *redis.StringCmd {
				return redis.NewStringCmd(ctx, "")
			},
			getSeatsFunc: func(ctx context.Context, showtimeID int, seatIDs []int) (*domain.ShowtimeSeats, error) {
				return &domain.ShowtimeSeats{
					Seats: []domain.Seat{
						{ID: 1, Row: 1, Col: 1, Type: "Standard", ExtraPrice: pgtype.Numeric{Int: decimal.NewFromFloat(0).BigInt(), Valid: true}},
					},
				}, nil
			},
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "the provided seat IDs don't match the available seats for the showtime",
		},
		{
			name:       "should handle concurrent seat locking failures",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: []int{1, 2, 3},
			},
			redisGetFunc: func(ctx context.Context, key string) *redis.StringCmd {
				return redis.NewStringCmd(ctx, "")
			},
			getSeatsFunc: func(ctx context.Context, showtimeID int, seatIDs []int) (*domain.ShowtimeSeats, error) {
				return &domain.ShowtimeSeats{
					Seats: []domain.Seat{
						{ID: 1, Row: 1, Col: 1, Type: "Standard", ExtraPrice: pgtype.Numeric{Int: decimal.NewFromFloat(0).BigInt(), Valid: true}},
						{ID: 2, Row: 1, Col: 2, Type: "Standard", ExtraPrice: pgtype.Numeric{Int: decimal.NewFromFloat(0).BigInt(), Valid: true}},
						{ID: 3, Row: 1, Col: 3, Type: "Standard", ExtraPrice: pgtype.Numeric{Int: decimal.NewFromFloat(0).BigInt(), Valid: true}},
					},
				}, nil
			},
			redisSetNXFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
				return redis.NewBoolCmd(ctx, false)
			},
			redisExecFunc: func(ctx context.Context, seatIDs []int) ([]redis.Cmder, error) {
				cmds := make([]redis.Cmder, len(seatIDs))
				for i := range seatIDs {
					if i == 0 {
						cmds[i] = redis.NewBoolResult(false, nil)
					} else {
						cmds[i] = redis.NewBoolResult(true, nil)
					}
				}
				return cmds, nil
			},
			wantStatus:     http.StatusConflict,
			wantErrMessage: ErrEditConflict,
		},
		{
			name:       "should successfully create cart with valid input",
			showtimeID: 1,
			input: api.CreateCartRequest{
				SeatIdList: []int{1, 2, 3},
			},
			getSeatsFunc: func(ctx context.Context, showtimeID int, seatIDs []int) (*domain.ShowtimeSeats, error) {
				return &domain.ShowtimeSeats{
					Seats: []domain.Seat{
						{ID: 1, Row: 1, Col: 1, Type: "Standard", ExtraPrice: pgtype.Numeric{Int: decimal.NewFromFloat(0).BigInt(), Valid: true}},
						{ID: 2, Row: 1, Col: 2, Type: "VIP", ExtraPrice: pgtype.Numeric{Int: decimal.NewFromFloat(15).BigInt(), Valid: true}},
						{ID: 3, Row: 1, Col: 3, Type: "Recliner", ExtraPrice: pgtype.Numeric{Int: decimal.NewFromFloat(10).BigInt(), Valid: true}},
					},
					Price: pgtype.Numeric{Int: decimal.NewFromFloat(50).BigInt(), Valid: true},
				}, nil
			},
			redisGetFunc: func(ctx context.Context, key string) *redis.StringCmd {
				return redis.NewStringCmd(ctx, "")
			},
			redisSetFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
				return redis.NewStatusCmd(ctx, "OK")
			},
			redisSAddFunc: func(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
				return redis.NewIntCmd(ctx, 1)
			},
			redisSetNXFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
				return redis.NewBoolCmd(ctx, true)
			},
			redisDelFunc: func(ctx context.Context, keys ...string) *redis.IntCmd {
				return redis.NewIntCmd(ctx, 1)
			},
			redisSRemFunc: func(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
				return redis.NewIntCmd(ctx, 1)
			},
			redisExecFunc: func(ctx context.Context, seatIDs []int) ([]redis.Cmder, error) {
				cmds := make([]redis.Cmder, len(seatIDs))
				for i := range seatIDs {
					cmds[i] = redis.NewBoolResult(true, nil)
				}
				return cmds, nil
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
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(func(a *application) {
				a.seatRepo = &mocks.MockSeatRepo{
					GetSeatsByShowtimeAndSeatIdsFunc: tt.getSeatsFunc,
				}
				a.sessionManager = scs.New()
				a.redis = &mocks.MockRedisClient{
					GetFunc: tt.redisGetFunc,
					TxPipelineFunc: func() redis.Pipeliner {
						return &mocks.MockTxPipeline{
							SetNXFunc: tt.redisSetNXFunc,
							SetFunc:   tt.redisSetFunc,
							SAddFunc:  tt.redisSAddFunc,
							ExecFunc: func(ctx context.Context) ([]redis.Cmder, error) {
								if tt.redisExecFunc != nil {
									return tt.redisExecFunc(ctx, tt.input.SeatIdList)
								}
								return nil, nil
							},
						}
					},
					DelFunc:  tt.redisDelFunc,
					SRemFunc: tt.redisSRemFunc,
				}
			})

			w, r := executeRequest(t, http.MethodPost, fmt.Sprintf("/showtimes/%d/cart", tt.showtimeID), tt.input)

			r = setupTestSession(t, app, r, 1)

			handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				app.CreateCartHandler(w, r, tt.showtimeID)
			}))
			handler = app.sessionManager.LoadAndSave(handler)
			handler.ServeHTTP(w, r)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("CreateCartHandler() status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantResponse != nil {
				var response api.CartResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				cmpOpts := cmpopts.IgnoreFields(api.Cart{}, "CartId")
				if diff := cmp.Diff(tt.wantResponse, &response, cmpOpts); diff != "" {
					t.Errorf("CreateCartHandler() response mismatch (-want +got):\n%s", diff)
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
