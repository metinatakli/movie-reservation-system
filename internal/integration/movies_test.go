package integration_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MovieTestSuite struct {
	BaseSuite
}

func TestMovieSuite(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	suite.Run(t, new(MovieTestSuite))
}

func (s *MovieTestSuite) TestGetMovies() {
	scenarios := []Scenario{
		{
			Name:           "returns empty list when no movies exist",
			Method:         "GET",
			URL:            "/movies",
			ExpectedStatus: 200,
			ExpectedResponse: `{
				"movies": [],
				"metadata": {
					"currentPage": 1,
					"firstPage": 1,
					"lastPage": 0,
					"pageSize": 10,
					"totalRecords": 0
				}
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/movies_down.sql")
			},
		},
		{
			Name:           "returns 422 for invalid page parameter",
			Method:         "GET",
			URL:            "/movies?page=-1",
			ExpectedStatus: 422,
			ExpectedResponse: `{
				"message": "One or more fields have invalid values",
				"validationErrors": [
					{"field": "Page", "issue": "must be at least 1"}
				]
			}`,
		},
		{
			Name:           "returns paginated movies",
			Method:         "GET",
			URL:            "/movies?page=2&pageSize=3",
			ExpectedStatus: 200,
			ExpectedResponse: `{
				"movies": [
					{"id": 4, "name": "Movie 4", "description": "Description 4", "posterUrl": "https://example.com/poster4.jpg", "releaseDate": "2025-01-04", "status": "NOW_SHOWING"},
					{"id": 5, "name": "Movie 5", "description": "Description 5", "posterUrl": "https://example.com/poster5.jpg", "releaseDate": "2025-01-05", "status": "NOW_SHOWING"},
					{"id": 6, "name": "Movie 6", "description": "Description 6", "posterUrl": "https://example.com/poster6.jpg", "releaseDate": "2025-01-06", "status": "NOW_SHOWING"}
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
				executeSQLFile(t, app.DB, "testdata/movies_down.sql")
				executeSQLFile(t, app.DB, "testdata/movies_up.sql")
			},
		},
		{
			Name:           "returns sorted movies by releaseDate desc",
			Method:         "GET",
			URL:            "/movies?sort=-release_date&page=1&pageSize=3",
			ExpectedStatus: 200,
			ExpectedResponse: `{
				"movies": [
					{"id": 7, "name": "Movie 7", "description": "Description 7", "posterUrl": "https://example.com/poster7.jpg", "releaseDate": "2025-01-07", "status": "NOW_SHOWING"},
					{"id": 6, "name": "Movie 6", "description": "Description 6", "posterUrl": "https://example.com/poster6.jpg", "releaseDate": "2025-01-06", "status": "NOW_SHOWING"},
					{"id": 5, "name": "Movie 5", "description": "Description 5", "posterUrl": "https://example.com/poster5.jpg", "releaseDate": "2025-01-05", "status": "NOW_SHOWING"}
				],
				"metadata": {
					"currentPage": 1,
					"firstPage": 1,
					"lastPage": 3,
					"pageSize": 3,
					"totalRecords": 7
				}
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/movies_down.sql")
				executeSQLFile(t, app.DB, "testdata/movies_up.sql")
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}

func (s *MovieTestSuite) TestShowMovieDetails() {
	scenarios := []Scenario{
		{
			Name:           "returns 400 for invalid movie ID",
			Method:         "GET",
			URL:            "/movies/0",
			ExpectedStatus: 400,
			ExpectedResponse: `{
				"message": "movie ID must be greater than zero"
			}`,
		},
		{
			Name:           "returns 404 when movie not found",
			Method:         "GET",
			URL:            "/movies/9999",
			ExpectedStatus: 404,
			ExpectedResponse: `{
				"message": "The requested resource not found"
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/movies_down.sql")
			},
		},
		{
			Name:           "successfully retrieves movie details",
			Method:         "GET",
			URL:            "/movies/1",
			ExpectedStatus: 200,
			ExpectedResponse: `{
				"id": 1,
				"name": "Movie 1",
				"posterUrl": "https://example.com/poster1.jpg",
				"releaseDate": "2025-01-01",
				"description": "Description 1",
				"runtime": 100,
				"genres": ["Action"],
				"language": "English",
				"director": "Director 1",
				"cast": ["Actor 1"],
				"rating": 7.0
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/movies_down.sql")
				executeSQLFile(t, app.DB, "testdata/movies_up.sql")
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}

func (s *MovieTestSuite) TestGetMovieShowtimes() {
	testDate := "2095-01-01"
	scenarios := []Scenario{
		{
			Name:             "returns 400 for invalid movie ID",
			Method:           "GET",
			URL:              fmt.Sprintf("/movies/0/showtimes?latitude=40.0&longitude=30.0&date=%s", testDate),
			ExpectedStatus:   400,
			ExpectedResponse: `{"message": "movie ID must be greater than zero"}`,
		},
		{
			Name:           "returns 422 for missing required params",
			Method:         "GET",
			URL:            "/movies/1/showtimes",
			ExpectedStatus: 422,
			ExpectedResponse: `{
				"message": "One or more fields have invalid values",
				"validationErrors": [
					{"field": "Latitude", "issue": "is required"},
					{"field": "Longitude", "issue": "is required"},
					{"field": "Date", "issue": "is required"}
				]
			}`,
		},
		{
			Name:             "returns 404 when movie not found",
			Method:           "GET",
			URL:              fmt.Sprintf("/movies/999/showtimes?latitude=40.0&longitude=30.0&date=%s", testDate),
			ExpectedStatus:   404,
			ExpectedResponse: `{"message": "The requested resource not found"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/showtimes_down.sql")
			},
		},
		{
			Name:           "successfully retrieves movie showtimes (all fields)",
			Method:         "GET",
			URL:            fmt.Sprintf("/movies/1/showtimes?latitude=40.0&longitude=30.0&date=%s", testDate),
			ExpectedStatus: 200,
			ExpectedResponse: `{
				"date": "2095-01-01",
				"theaters": [
					{
						"id": 1,
						"name": "Test Theater 1",
						"address": "123 Main St",
						"city": "Test City",
						"district": "Central",
						"distance": 0,
						"amenities": [],
						"halls": [
							{
								"id": 1,
								"name": "Hall 1A",
								"amenities": [
									{"id": 1, "name": "IMAX", "description": "Large-format screen"}
								],
								"showtimes": [
									{"id": 1, "startTime": "10:00", "startDateTime": "2095-01-01T10:00:00Z", "price": 10.0, "status": "AVAILABLE"},
									{"id": 2, "startTime": "14:00", "startDateTime": "2095-01-01T14:00:00Z", "price": 12.0, "status": "AVAILABLE"}
								]
							},
							{
								"id": 2,
								"name": "Hall 1B",
								"amenities": [
									{"id": 2, "name": "Dolby Atmos", "description": "Immersive sound system"}
								],
								"showtimes": [
									{"id": 3, "startTime": "10:00", "startDateTime": "2095-01-01T10:00:00Z", "price": 11.0, "status": "AVAILABLE"},
									{"id": 4, "startTime": "14:00", "startDateTime": "2095-01-01T14:00:00Z", "price": 13.0, "status": "AVAILABLE"}
								]
							}
						]
					},
					{
						"id": 2,
						"name": "Test Theater 2",
						"address": "456 Side St",
						"city": "Test City",
						"district": "North",
						"distance": 0,
						"amenities": [],
						"halls": [
							{
								"id": 3,
								"name": "Hall 2A",
								"amenities": [
									{"id": 1, "name": "IMAX", "description": "Large-format screen"}
								],
								"showtimes": [
									{"id": 5, "startTime": "10:00", "startDateTime": "2095-01-01T10:00:00Z", "price": 10.5, "status": "AVAILABLE"},
									{"id": 6, "startTime": "14:00", "startDateTime": "2095-01-01T14:00:00Z", "price": 12.5, "status": "AVAILABLE"}
								]
							},
							{
								"id": 4,
								"name": "Hall 2B",
								"amenities": [
									{"id": 2, "name": "Dolby Atmos", "description": "Immersive sound system"}
								],
								"showtimes": [
									{"id": 7, "startTime": "10:00", "startDateTime": "2095-01-01T10:00:00Z", "price": 11.5, "status": "AVAILABLE"},
									{"id": 8, "startTime": "14:00", "startDateTime": "2095-01-01T14:00:00Z", "price": 13.5, "status": "AVAILABLE"}
								]
							}
						]
					}
				],
				"metadata": {
					"currentPage": 1,
					"firstPage": 1,
					"lastPage": 1,
					"pageSize": 10,
					"totalRecords": 2
				}
			}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/showtimes_down.sql")
				executeSQLFile(t, app.DB, "testdata/movies_down.sql")

				executeSQLFile(t, app.DB, "testdata/movies_up.sql")
				executeSQLFile(t, app.DB, "testdata/showtimes_up.sql")
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}
