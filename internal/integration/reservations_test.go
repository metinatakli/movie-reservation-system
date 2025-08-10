package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ReservationTestSuite struct {
	BaseSuite
}

func TestReservationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	suite.Run(t, new(ReservationTestSuite))
}

func (s *ReservationTestSuite) TestGetReservationsOfUserHandler() {
	cookies := s.app.authenticatedUserCookies(s.T())

	scenarios := []Scenario{
		{
			Name:             "returns 401 if user is not authenticated",
			Method:           "GET",
			URL:              "/users/me/reservations",
			ExpectedStatus:   http.StatusUnauthorized,
			ExpectedResponse: `{"message": "You must be authenticated to access this resource"}`,
		},
		{
			Name:           "returns 422 for invalid page parameter",
			Method:         "GET",
			URL:            "/users/me/reservations?page=0",
			Cookies:        cookies,
			ExpectedStatus: http.StatusUnprocessableEntity,
			ExpectedResponse: `{
				"message": "One or more fields have invalid values",
				"validationErrors": [
					{"field": "Page", "issue": "must be at least 1"}
				]
			}`,
		},
		{
			Name:           "returns empty list when user has no reservations",
			Method:         "GET",
			URL:            "/users/me/reservations",
			Cookies:        cookies,
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
				"reservations": [],
				"metadata": {
					"currentPage": 1,
					"firstPage": 1,
					"lastPage": 0,
					"pageSize": 10,
					"totalRecords": 0
				}
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupReservationTestState(t, app)
			},
		},
		{
			Name:           "returns paginated reservations",
			Method:         "GET",
			URL:            "/users/me/reservations?page=2&pageSize=3",
			Cookies:        cookies,
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
				"reservations": [
					{ "id": 4, "movieTitle": "Movie 1", "moviePosterUrl": "https://example.com/poster1.jpg", "hallName": "Hall 1B", "theaterName": "Test Theater 1", "date": "2095-01-01T17:00:00+03:00" },
					{ "id": 5, "movieTitle": "Movie 1", "moviePosterUrl": "https://example.com/poster1.jpg", "hallName": "Hall 2A", "theaterName": "Test Theater 2", "date": "2095-01-01T13:00:00+03:00" },
					{ "id": 6, "movieTitle": "Movie 1", "moviePosterUrl": "https://example.com/poster1.jpg", "hallName": "Hall 2A", "theaterName": "Test Theater 2", "date": "2095-01-01T17:00:00+03:00" }
				],
				"metadata": {
					"currentPage": 2,
					"firstPage": 1,
					"lastPage": 3,
					"pageSize": 3,
					"totalRecords": 7
				}
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupReservationTestState(t, app)
				executeSQLFile(t, app.DB, "testdata/reservations_down.sql")
				executeSQLFile(t, app.DB, "testdata/reservations_up.sql")
			},
		},
		{
			Name:           "returns the last page which may not be full",
			Method:         "GET",
			URL:            "/users/me/reservations?page=3&pageSize=3",
			Cookies:        cookies,
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
				"reservations": [
					{ "id": 7, "movieTitle": "Movie 1", "moviePosterUrl": "https://example.com/poster1.jpg", "hallName": "Hall 2B", "theaterName": "Test Theater 2", "date": "2095-01-01T13:00:00+03:00" }
				],
				"metadata": {
					"currentPage": 3,
					"firstPage": 1,
					"lastPage": 3,
					"pageSize": 3,
					"totalRecords": 7
				}
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupReservationTestState(t, app)
				executeSQLFile(t, app.DB, "testdata/reservations_down.sql")
				executeSQLFile(t, app.DB, "testdata/reservations_up.sql")
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}

func (s *ReservationTestSuite) TestGetUserReservationById() {
	cookies := s.app.authenticatedUserCookies(s.T())

	scenarios := []Scenario{
		{
			Name:             "returns 401 if user is not authenticated",
			Method:           "GET",
			URL:              "/users/me/reservations/1",
			ExpectedStatus:   http.StatusUnauthorized,
			ExpectedResponse: `{"message": "You must be authenticated to access this resource"}`,
		},
		{
			Name:             "returns 400 for invalid reservation ID",
			Method:           "GET",
			URL:              "/users/me/reservations/0",
			Cookies:          cookies,
			ExpectedStatus:   http.StatusBadRequest,
			ExpectedResponse: `{"message": "reservation id must be greater than zero"}`,
		},
		{
			Name:             "returns 404 for a reservation that does not exist",
			Method:           "GET",
			URL:              "/users/me/reservations/999",
			Cookies:          cookies,
			ExpectedStatus:   http.StatusNotFound,
			ExpectedResponse: `{"message": "The requested resource not found"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/reservation_details.sql")
			},
		},
		{
			Name:             "returns 404 if user tries to access another user's reservation",
			Method:           "GET",
			URL:              "/users/me/reservations/2",
			Cookies:          cookies,
			ExpectedStatus:   http.StatusNotFound,
			ExpectedResponse: `{"message": "The requested resource not found"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/reservation_details.sql")
			},
		},
		{
			Name:           "successfully returns reservation details for the authenticated user",
			Method:         "GET",
			URL:            "/users/me/reservations/1",
			Cookies:        cookies,
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
				"id": 1,
				"movieTitle": "The Go Story",
				"moviePosterUrl": "https://example.com/poster-go.jpg",
				"theaterName": "Grand Cinema",
				"hallName": "Hall A",
				"totalPrice": "25",
				"date": "2095-05-10T23:00:00+03:00",
				"seats": [
					{"row": 3, "column": 5, "type": "Standard"},
					{"row": 3, "column": 6, "type": "Standard"}
				],
				"theaterAmenities": [
					{"id": 1, "name": "Cafe", "description": "Serves coffee and snacks."},
					{"id": 2, "name": "Parking", "description": "On-site parking available."}
				],
				"hallAmenities": [
					{"id": 3, "name": "IMAX", "description": "Large-format screen."},
					{"id": 4, "name": "Dolby Atmos", "description": "Immersive sound system."}
				]
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/reservation_details.sql")
			},
		},
		{
			Name:           "successfully returns reservation with empty amenities",
			Method:         "GET",
			URL:            "/users/me/reservations/1",
			Cookies:        cookies,
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
				"id": 1,
				"movieTitle": "The Go Story",
				"moviePosterUrl": "https://example.com/poster-go.jpg",
				"theaterName": "Grand Cinema",
				"hallName": "Hall A",
				"totalPrice": "25",
				"date": "2095-05-10T23:00:00+03:00",
				"seats": [
					{"row": 3, "column": 5, "type": "Standard"},
					{"row": 3, "column": 6, "type": "Standard"}
				],
				"theaterAmenities": [],
				"hallAmenities": []
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/reservation_details.sql")
				// Manually delete the amenities to test the empty slice case
				_, err := app.DB.Exec(context.Background(), "TRUNCATE TABLE theater_amenities, hall_amenities")
				require.NoError(t, err)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}

func setupReservationTestState(t testing.TB, app *TestApp) {
	t.Helper()

	executeSQLFile(t, app.DB, "testdata/movies_down.sql")
	executeSQLFile(t, app.DB, "testdata/showtimes_down.sql")
	executeSQLFile(t, app.DB, "testdata/users_down.sql")
	executeSQLFile(t, app.DB, "testdata/seats_down.sql")

	executeSQLFile(t, app.DB, "testdata/movies_up.sql")
	executeSQLFile(t, app.DB, "testdata/showtimes_up.sql")
	executeSQLFile(t, app.DB, "testdata/users_up.sql")
	executeSQLFile(t, app.DB, "testdata/seats_up.sql")
}
