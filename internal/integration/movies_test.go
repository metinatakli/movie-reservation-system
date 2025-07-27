package integration_test

import (
	"fmt"
	"testing"
	"time"

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
				truncateMovies(t, app.DB)
			},
		},
		{
			Name:           "returns single movie",
			Method:         "GET",
			URL:            "/movies",
			ExpectedStatus: 200,
			ExpectedResponse: fmt.Sprintf(`{
				"movies": [
					{
						"id": 1,
						"name": "%s",
						"description": "%s",
						"posterUrl": "%s",
						"releaseDate": "%s",
						"status": "NOW_SHOWING"
					}
				],
				"metadata": {
					"currentPage": 1,
					"firstPage": 1,
					"lastPage": 1,
					"pageSize": 10,
					"totalRecords": 1
				}
			}`,
				TestMovieTitle,
				TestMovieDescription,
				TestMoviePosterUrl,
				TestMovieReleaseDate,
			),
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateMovies(t, app.DB)
				insertTestMovie(t, app.DB, defaultTestMovie())
			},
		},
		{
			Name:           "returns multiple movies",
			Method:         "GET",
			URL:            "/movies",
			ExpectedStatus: 200,
			ExpectedResponse: fmt.Sprintf(`{
				"movies": [
					{"id": 1, "name": "%s", "description": "%s", "posterUrl": "%s", "releaseDate": "%s", "status": "NOW_SHOWING"},
					{"id": 2, "name": "Another Movie", "description": "Another description.", "posterUrl": "https://example.com/another.jpg", "releaseDate": "%s", "status": "COMING_SOON"}
				],
				"metadata": {
					"currentPage": 1,
					"firstPage": 1,
					"lastPage": 1,
					"pageSize": 10,
					"totalRecords": 2
				}
			}`,
				TestMovieTitle,
				TestMovieDescription,
				TestMoviePosterUrl,
				TestMovieReleaseDate,
				time.Now().AddDate(50, 0, 0).Format("2006-01-02"),
			),
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				truncateMovies(t, app.DB)
				insertTestMovie(t, app.DB, defaultTestMovie())

				m := defaultTestMovie()
				m.Title = "Another Movie"
				m.Description = "Another description."
				m.PosterUrl = "https://example.com/another.jpg"
				m.ReleaseDate, _ = time.Parse("2006-01-02", time.Now().AddDate(50, 0, 0).Format("2006-01-02"))
				insertTestMovie(t, app.DB, m)
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
				truncateMovies(t, app.DB)
				executeSQLFile(t, app.DB, "testdata/movies.sql")
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
				truncateMovies(t, app.DB)
				executeSQLFile(t, app.DB, "testdata/movies.sql")
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
				truncateMovies(t, app.DB)
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
				truncateMovies(t, app.DB)
				executeSQLFile(t, app.DB, "testdata/movies.sql")
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Run(s.T(), s.app)
	}
}
