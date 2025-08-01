package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type SeatMapTestSuite struct {
	BaseSuite
}

func TestSeatMapSuite(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	suite.Run(t, new(SeatMapTestSuite))
}

// TODO: Requests and responses can be moved to under testdata folder as .json files This will help to keep the test code cleaner.
func (s *SeatMapTestSuite) TestGetSeatMapByShowtime() {
	scenarios := []Scenario{
		{
			Name:             "returns 400 for invalid showtime ID",
			Method:           "GET",
			URL:              "/showtimes/0/seat-map",
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: `{"message": "showtime ID must be greater than zero"}`,
		},
		{
			Name:             "returns 404 for non-existent showtime",
			Method:           "GET",
			URL:              "/showtimes/999/seat-map",
			ExpectedStatus:   http.StatusNotFound,
			ExpectedResponse: `{"message": "The requested resource not found"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseSeatMapState(t, app)
			},
		},
		{
			Name:           "returns seat map with all seats available",
			Method:         "GET",
			URL:            "/showtimes/1/seat-map",
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
				"theaterId": 1,
				"theaterName": "Test Theater 1",
				"hallId": 1,
				"showtimeId": 1,
				"seatRows": [
					{
						"row": 1,
						"seats": [
							{"id": 1, "row": 1, "column": 1, "extraPrice": "0", "type": "Standard", "available": true},
							{"id": 2, "row": 1, "column": 2, "extraPrice": "0", "type": "Standard", "available": true}
						]
					},
					{
						"row": 2,
						"seats": [
							{"id": 3, "row": 2, "column": 1, "extraPrice": "10.49", "type": "VIP", "available": true},
							{"id": 4, "row": 2, "column": 2, "extraPrice": "8.49", "type": "Recliner", "available": true}
						]
					}
				]
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseSeatMapState(t, app)
			},
		},
		{
			Name:           "returns seat map with reserved seats unavailable",
			Method:         "GET",
			URL:            "/showtimes/1/seat-map",
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
				"theaterId": 1,
				"theaterName": "Test Theater 1",
				"hallId": 1,
				"showtimeId": 1,
				"seatRows": [
					{
						"row": 1,
						"seats": [
							{"id": 1, "row": 1, "column": 1, "extraPrice": "0", "type": "Standard", "available": true},
							{"id": 2, "row": 1, "column": 2, "extraPrice": "0", "type": "Standard", "available": false}
						]
					},
					{
						"row": 2,
						"seats": [
							{"id": 3, "row": 2, "column": 1, "extraPrice": "10.49", "type": "VIP", "available": true},
							{"id": 4, "row": 2, "column": 2, "extraPrice": "8.49", "type": "Recliner", "available": true}
						]
					}
				]
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseSeatMapState(t, app)
				executeSQLFile(t, app.DB, "testdata/seat_reservations_up.sql")
			},
		},
		{
			Name:           "returns seat map with locked seats unavailable",
			Method:         "GET",
			URL:            "/showtimes/1/seat-map",
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
				"theaterId": 1,
				"theaterName": "Test Theater 1",
				"hallId": 1,
				"showtimeId": 1,
				"seatRows": [
					{
						"row": 1,
						"seats": [
							{"id": 1, "row": 1, "column": 1, "extraPrice": "0", "type": "Standard", "available": true},
							{"id": 2, "row": 1, "column": 2, "extraPrice": "0", "type": "Standard", "available": true}
						]
					},
					{
						"row": 2,
						"seats": [
							{"id": 3, "row": 2, "column": 1, "extraPrice": "10.49", "type": "VIP", "available": false},
							{"id": 4, "row": 2, "column": 2, "extraPrice": "8.49", "type": "Recliner", "available": true}
						]
					}
				]
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseSeatMapState(t, app)
				lockSeatInCache(t, app.RedisClient, 1, 3)
			},
		},
		{
			Name:           "returns seat map with both locked and reserved seats unavailable",
			Method:         "GET",
			URL:            "/showtimes/1/seat-map",
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
				"theaterId": 1,
				"theaterName": "Test Theater 1",
				"hallId": 1,
				"showtimeId": 1,
				"seatRows": [
					{
						"row": 1,
						"seats": [
							{"id": 1, "row": 1, "column": 1, "extraPrice": "0", "type": "Standard", "available": true},
							{"id": 2, "row": 1, "column": 2, "extraPrice": "0", "type": "Standard", "available": false}
						]
					},
					{
						"row": 2,
						"seats": [
							{"id": 3, "row": 2, "column": 1, "extraPrice": "10.49", "type": "VIP", "available": false},
							{"id": 4, "row": 2, "column": 2, "extraPrice": "8.49", "type": "Recliner", "available": true}
						]
					}
				]
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseSeatMapState(t, app)
				executeSQLFile(t, app.DB, "testdata/seat_reservations_up.sql")
				lockSeatInCache(t, app.RedisClient, 1, 3)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}

// setupBaseSeatMapState prepares the database with all necessary data for seat map tests.
func setupBaseSeatMapState(t testing.TB, app *TestApp) {
	t.Helper()

	executeSQLFile(t, app.DB, "testdata/seats_down.sql")
	executeSQLFile(t, app.DB, "testdata/showtimes_down.sql")
	executeSQLFile(t, app.DB, "testdata/seat_reservations_down.sql")
	flushAllCache(t, app.RedisClient)

	executeSQLFile(t, app.DB, "testdata/movies_up.sql")
	executeSQLFile(t, app.DB, "testdata/showtimes_up.sql")
	executeSQLFile(t, app.DB, "testdata/seats_up.sql")
}
