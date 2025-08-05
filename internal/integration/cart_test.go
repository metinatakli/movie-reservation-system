package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CartTestSuite struct {
	BaseSuite
}

func TestCartSuite(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	suite.Run(t, new(CartTestSuite))
}

func (s *CartTestSuite) TestCreateCartHandler() {
	cookies := s.app.authenticatedUserCookies(s.T())

	scenarios := []Scenario{
		{
			Name:             "returns 400 for invalid showtime ID",
			Method:           "POST",
			URL:              "/showtimes/0/cart",
			Body:             strings.NewReader(`{"seatIdList": [1, 2]}`),
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: `{"message": "showtime ID must be greater than zero"}`,
		},
		{
			Name:           "returns 422 for invalid request body (empty seat list)",
			Method:         "POST",
			URL:            "/showtimes/1/cart",
			Body:           strings.NewReader(`{"seatIdList": []}`),
			Cookies:        cookies,
			ExpectedStatus: http.StatusUnprocessableEntity,
			ExpectedResponse: `{
				"message": "One or more fields have invalid values",
				"validationErrors": [
					{"field": "SeatIdList", "issue": "must contain at least 1 items"}
				]
			}`,
		},
		{
			Name:             "returns 400 if a cart already exists in the session",
			Method:           "POST",
			URL:              "/showtimes/1/cart",
			Body:             strings.NewReader(`{"seatIdList": [1]}`),
			Cookies:          cookies,
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: `{"message": "cannot create new cart if a cart already exists in session"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseCreateCartHandlerState(t, app)

				// Pre-populate redis with an existing cart for this session
				key := fmt.Sprintf("cart:%s", cookies[0].Value)
				err := app.RedisClient.Set(context.Background(), key, "existing-cart-id", 10*time.Minute).Err()
				if err != nil {
					t.Fatalf("failed to set up redis for test: %v", err)
				}
			},
		},
		{
			Name:             "returns 409 if a selected seat is already reserved in the database",
			Method:           "POST",
			URL:              "/showtimes/1/cart",
			Body:             strings.NewReader(`{"seatIdList": [2, 3]}`), // Seat 2 is already reserved
			Cookies:          cookies,
			ExpectedStatus:   http.StatusConflict,
			ExpectedResponse: `{"message": "some of the selected seats are already reserved"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseCreateCartHandlerState(t, app)
				executeSQLFile(t, app.DB, "testdata/seat_reservations_up.sql")
			},
		},
		{
			Name:             "returns 404 if a selected seat does not exist for the showtime",
			Method:           "POST",
			URL:              "/showtimes/1/cart",
			Body:             strings.NewReader(`{"seatIdList": [1, 99]}`),
			Cookies:          cookies,
			ExpectedStatus:   http.StatusNotFound,
			ExpectedResponse: `{"message": "The requested resource not found"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseCreateCartHandlerState(t, app)
			},
		},
		{
			Name:             "returns 409 if a selected seat is already locked by another session",
			Method:           "POST",
			URL:              "/showtimes/1/cart",
			Body:             strings.NewReader(`{"seatIdList": [3, 4]}`), // We will lock seat 3
			Cookies:          cookies,
			ExpectedStatus:   http.StatusConflict,
			ExpectedResponse: `{"message": "some of the selected seats are already reserved"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseCreateCartHandlerState(t, app)
				lockSeatInCache(t, app.RedisClient, 1, 3, "another-session-id")
			},
		},
		{
			Name:           "successfully creates a cart and locks seats",
			Method:         "POST",
			URL:            "/showtimes/1/cart",
			Body:           strings.NewReader(`{"seatIdList": [1, 4]}`),
			Cookies:        cookies,
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
				"cart": {
					"showtimeId": 1,
					"movieName": "Movie 1",
					"theaterName": "Test Theater 1",
					"hallName": "Hall 1A",
					"showtimeDate": "Sat, 01 Jan 2095 13:00:00 +03",
					"seats": [
						{"id": 1, "row": 1, "column": 1, "type": "Standard", "price": "0"},
						{"id": 4, "row": 2, "column": 2, "type": "Recliner", "price": "8.49"}
					],
					"holdTime": 600,
					"basePrice": "10",
					"totalPrice": "28.49"
				}
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseCreateCartHandlerState(t, app)
			},
			AfterTestFunc: func(t testing.TB, app *TestApp, res *http.Response) {
				// Verify that locks were created in Redis
				ctx := context.Background()
				key1 := "seat_lock:1:1"
				key2 := "seat_lock:1:4"

				session1, err := app.RedisClient.Get(ctx, key1).Result()
				if err != nil || session1 != cookies[0].Value {
					t.Errorf("expected seat 1 to be locked by '%s', got '%s', err: %v", cookies[0].Value, session1, err)
				}
				session2, err := app.RedisClient.Get(ctx, key2).Result()
				if err != nil || session2 != cookies[0].Value {
					t.Errorf("expected seat 4 to be locked by '%s', got '%s', err: %v", cookies[0].Value, session2, err)
				}
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}

func (s *CartTestSuite) TestDeleteCartHandler() {
	cookies := s.app.authenticatedUserCookies(s.T())

	scenarios := []Scenario{
		{
			Name:             "returns 400 for invalid showtime ID",
			Method:           "DELETE",
			URL:              "/showtimes/0/cart",
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: `{"message": "showtime ID must be greater than zero"}`,
		},
		{
			Name:             "returns 404 if no cart exists for the session",
			Method:           "DELETE",
			URL:              "/showtimes/1/cart",
			Cookies:          cookies,
			ExpectedStatus:   http.StatusNotFound,
			ExpectedResponse: `{"message": "The requested resource not found"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseCreateCartHandlerState(t, app)
			},
		},
		{
			Name:             "returns 404 if session points to an expired/non-existent cart object",
			Method:           "DELETE",
			URL:              "/showtimes/1/cart",
			Cookies:          cookies,
			ExpectedStatus:   http.StatusNotFound,
			ExpectedResponse: `{"message": "The requested resource not found"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseCreateCartHandlerState(t, app)

				cartSessionKey := fmt.Sprintf("cart:%s", cookies[0].Value)
				err := app.RedisClient.Set(context.Background(), cartSessionKey, "dangling-cart-id", 10*time.Minute).Err()
				require.NoError(t, err)
			},
		},
		{
			Name:             "returns 404 if the showtime ID in the URL does not match the cart's showtime ID",
			Method:           "DELETE",
			URL:              "/showtimes/999/cart",
			Cookies:          cookies,
			ExpectedStatus:   http.StatusNotFound,
			ExpectedResponse: `{"message": "The requested resource not found"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseCreateCartHandlerState(t, app)
				createTestCartInCache(s.T(), app, cookies[0].Value, 1, []domain.CartSeat{{Id: 1}})
			},
		},
		{
			Name:           "returns 204 when successfully deletes a cart and all associated keys",
			Method:         "DELETE",
			URL:            "/showtimes/1/cart",
			Cookies:        cookies,
			ExpectedStatus: http.StatusNoContent,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseCreateCartHandlerState(t, app)
				createTestCartInCache(t, app, cookies[0].Value, 1, []domain.CartSeat{{Id: 1}, {Id: 4}})
			},
			AfterTestFunc: func(t testing.TB, app *TestApp, res *http.Response) {
				ctx := context.Background()

				// Verify all keys are gone.
				keysToCheck := []string{
					fmt.Sprintf("cart:%s", cookies[0].Value),
					"seat_lock:1:1",
					"seat_lock:1:4",
				}

				count, err := app.RedisClient.Exists(ctx, keysToCheck...).Result()
				require.NoError(t, err)
				require.Equal(t, int64(0), count, "expected cart and lock keys to be deleted")

				// Verify the seat set is empty
				seatLockSetKey := "seat_locks:1"
				members, err := app.RedisClient.SMembers(ctx, seatLockSetKey).Result()
				require.NoError(t, err)
				require.Empty(t, members, "expected seat set to be empty")
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}

// setupBaseCreateCartHandlerState prepares the database and cache with all necessary data for create cart handler tests.
func setupBaseCreateCartHandlerState(t testing.TB, app *TestApp) {
	t.Helper()

	executeSQLFile(t, app.DB, "testdata/seats_down.sql")
	executeSQLFile(t, app.DB, "testdata/showtimes_down.sql")
	executeSQLFile(t, app.DB, "testdata/seat_reservations_down.sql")
	truncateCartRelatedCache(t, app.RedisClient)

	executeSQLFile(t, app.DB, "testdata/movies_up.sql")
	executeSQLFile(t, app.DB, "testdata/showtimes_up.sql")
	executeSQLFile(t, app.DB, "testdata/seats_up.sql")
}
