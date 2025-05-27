package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/metinatakli/movie-reservation-system/internal/mocks"
	"github.com/metinatakli/movie-reservation-system/internal/validator"
	"github.com/oapi-codegen/runtime/types"
)

func TestGetMovies(t *testing.T) {
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	tests := []struct {
		name           string
		params         api.GetMoviesParams
		url            string
		getAllFunc     func(context.Context, domain.Pagination) ([]*domain.Movie, *domain.Metadata, error)
		wantStatus     int
		wantErrMessage string
		wantResponse   *api.MovieListResponse
	}{
		{
			name:   "successful retrieval with default parameters",
			params: api.GetMoviesParams{},
			url:    "/movies",
			getAllFunc: func(ctx context.Context, filters domain.Pagination) ([]*domain.Movie, *domain.Metadata, error) {
				movies := []*domain.Movie{
					{
						ID:          1,
						Title:       "Movie 1",
						Description: "Description 1",
						PosterUrl:   "http://example.com/poster1.jpg",
						ReleaseDate: yesterday,
					},
					{
						ID:          2,
						Title:       "Movie 2",
						Description: "Description 2",
						PosterUrl:   "http://example.com/poster2.jpg",
						ReleaseDate: tomorrow,
					},
				}
				metadata := &domain.Metadata{
					CurrentPage:  1,
					FirstPage:    1,
					LastPage:     1,
					PageSize:     10,
					TotalRecords: 2,
				}
				return movies, metadata, nil
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.MovieListResponse{
				Movies: []api.MovieSummary{
					{
						Id:          1,
						Name:        "Movie 1",
						Description: "Description 1",
						PosterUrl:   "http://example.com/poster1.jpg",
						ReleaseDate: types.Date{Time: yesterday},
						Status:      api.NOWSHOWING,
					},
					{
						Id:          2,
						Name:        "Movie 2",
						Description: "Description 2",
						PosterUrl:   "http://example.com/poster2.jpg",
						ReleaseDate: types.Date{Time: tomorrow},
						Status:      api.COMINGSOON,
					},
				},
				Metadata: &api.Metadata{
					CurrentPage:  1,
					FirstPage:    1,
					LastPage:     1,
					PageSize:     10,
					TotalRecords: 2,
				},
			},
		},
		{
			name: "successful retrieval with custom parameters",
			params: api.GetMoviesParams{
				Page:     ptr(2),
				PageSize: ptr(5),
				Sort:     ptr("title"),
				Term:     ptr("action"),
			},
			url: "/movies?page=2&pageSize=5&sort=title&term=action",
			getAllFunc: func(ctx context.Context, filters domain.Pagination) ([]*domain.Movie, *domain.Metadata, error) {
				movies := []*domain.Movie{
					{
						ID:          3,
						Title:       "Action Movie",
						Description: "Action packed",
						PosterUrl:   "http://example.com/action.jpg",
						ReleaseDate: yesterday,
					},
				}
				metadata := &domain.Metadata{
					CurrentPage:  2,
					FirstPage:    1,
					LastPage:     3,
					PageSize:     5,
					TotalRecords: 11,
				}
				return movies, metadata, nil
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.MovieListResponse{
				Movies: []api.MovieSummary{
					{
						Id:          3,
						Name:        "Action Movie",
						Description: "Action packed",
						PosterUrl:   "http://example.com/action.jpg",
						ReleaseDate: types.Date{Time: yesterday},
						Status:      api.NOWSHOWING,
					},
				},
				Metadata: &api.Metadata{
					CurrentPage:  2,
					FirstPage:    1,
					LastPage:     3,
					PageSize:     5,
					TotalRecords: 11,
				},
			},
		},
		{
			name: "validation error - negative page",
			params: api.GetMoviesParams{
				Page: ptr(-1),
			},
			url:            "/movies?page=-1",
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: fmt.Sprintf(validator.ErrMinValue, "1"),
		},
		{
			name: "validation error - page size too large",
			params: api.GetMoviesParams{
				PageSize: ptr(1000),
			},
			url:            "/movies?pageSize=1000",
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: fmt.Sprintf(validator.ErrMaxValue, "100"),
		},
		{
			name: "validation error - invalid sort parameter",
			params: api.GetMoviesParams{
				Sort: ptr("id; DROP TABLE movies; --"),
			},
			url:            "/movies?sort=invalid_column",
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: fmt.Sprintf(validator.ErrOneOf, "id -id release_date -release_date title -title duration -duration"),
		},
		{
			name: "validation error - term too long",
			params: api.GetMoviesParams{
				Term: ptr(strings.Repeat("a", 256)),
			},
			url:            "/movies?term=" + strings.Repeat("a", 256),
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: fmt.Sprintf(validator.ErrMaxLength, "50"),
		},
		{
			name:   "database error",
			params: api.GetMoviesParams{},
			url:    "/movies",
			getAllFunc: func(ctx context.Context, filters domain.Pagination) ([]*domain.Movie, *domain.Metadata, error) {
				return nil, nil, fmt.Errorf("database connection error")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name:   "empty result",
			params: api.GetMoviesParams{},
			url:    "/movies",
			getAllFunc: func(ctx context.Context, filters domain.Pagination) ([]*domain.Movie, *domain.Metadata, error) {
				return []*domain.Movie{}, &domain.Metadata{
					CurrentPage:  1,
					FirstPage:    1,
					LastPage:     1,
					PageSize:     10,
					TotalRecords: 0,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.MovieListResponse{
				Movies: []api.MovieSummary{},
				Metadata: &api.Metadata{
					CurrentPage:  1,
					FirstPage:    1,
					LastPage:     1,
					PageSize:     10,
					TotalRecords: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(func(a *application) {
				a.movieRepo = &mocks.MockMovieRepo{
					GetAllFunc: tt.getAllFunc,
				}
			})

			w, r := executeRequest(t, http.MethodGet, tt.url, nil)

			app.GetMovies(w, r, tt.params)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("GetMovies() status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantResponse != nil {
				var response api.MovieListResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if diff := cmp.Diff(tt.wantResponse, &response); diff != "" {
					t.Errorf("GetMovies() response mismatch (-want +got):\n%s", diff)
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

func TestShowMovieDetails(t *testing.T) {
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)

	tests := []struct {
		name           string
		id             int
		getByIdFunc    func(context.Context, int) (*domain.Movie, error)
		wantStatus     int
		wantResponse   *api.MovieDetailsResponse
		wantErrMessage string
	}{
		{
			name: "successful retrieval",
			id:   1,
			getByIdFunc: func(ctx context.Context, id int) (*domain.Movie, error) {
				return &domain.Movie{
					ID:          1,
					Title:       "Test Movie",
					Description: "A great movie",
					PosterUrl:   "http://example.com/poster.jpg",
					ReleaseDate: yesterday,
					Director:    "John Doe",
					CastMembers: []string{"Actor One", "Actor Two"},
					Rating: pgtype.Numeric{
						Int:   big.NewInt(85),
						Exp:   -1,
						Valid: true,
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.MovieDetailsResponse{
				Id:          1,
				Name:        "Test Movie",
				Description: "A great movie",
				PosterUrl:   "http://example.com/poster.jpg",
				ReleaseDate: types.Date{Time: yesterday},
				Director:    "John Doe",
				Cast:        []string{"Actor One", "Actor Two"},
				Rating:      ptr(float32(8.5)),
			},
		},
		{
			name: "movie not found",
			id:   2,
			getByIdFunc: func(ctx context.Context, id int) (*domain.Movie, error) {
				return nil, domain.ErrRecordNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:           "invalid ID",
			id:             0,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "movie ID must be greater than zero",
		},
		{
			name: "server error",
			id:   3,
			getByIdFunc: func(ctx context.Context, id int) (*domain.Movie, error) {
				return nil, fmt.Errorf("database error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(func(a *application) {
				a.movieRepo = &mocks.MockMovieRepo{
					GetByIdFunc: tt.getByIdFunc,
				}
			})

			w, r := executeRequest(t, http.MethodGet, fmt.Sprintf("/movies/%d", tt.id), nil)

			app.ShowMovieDetails(w, r, tt.id)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("ShowMovieDetails() status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantResponse != nil {
				var response api.MovieDetailsResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if diff := cmp.Diff(tt.wantResponse, &response); diff != "" {
					t.Errorf("ShowMovieDetails() response mismatch (-want +got):\n%s", diff)
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

func TestGetMovieShowtimes(t *testing.T) {
	now := time.Now()
	futureTime := now.Add(24 * time.Hour)
	pastTime := now.Add(-24 * time.Hour)

	tests := []struct {
		name            string
		id              int
		params          api.GetMovieShowtimesParams
		url             string
		getTheatersFunc func(context.Context, int, time.Time, float64, float64, domain.Pagination) ([]domain.Theater, *domain.Metadata, error)
		wantStatus      int
		wantErrMessage  string
		wantResponse    *api.MovieShowtimesResponse
	}{
		{
			name: "successful retrieval",
			id:   1,
			params: api.GetMovieShowtimesParams{
				Date:      "2024-03-20",
				Latitude:  39.990067,
				Longitude: 32.643482,
			},
			url: "/movies/1/showtimes?date=2024-03-20&latitude=39.990067&longitude=32.643482",
			getTheatersFunc: func(ctx context.Context, movieID int, date time.Time, lon, lat float64, pagination domain.Pagination) (
				[]domain.Theater,
				*domain.Metadata,
				error,
			) {
				theaters := []domain.Theater{
					{
						ID:       1,
						Name:     "Test Theater",
						Address:  "123 Test St",
						City:     "Test City",
						District: "Test District",
						Distance: 2.5,
						Amenities: []domain.Amenity{
							{ID: 1, Name: "IMAX", Description: "IMAX Screen"},
						},
						Halls: []domain.Hall{
							{
								ID:   1,
								Name: "Hall 1",
								Amenities: []domain.Amenity{
									{ID: 2, Name: "Dolby", Description: "Dolby Sound"},
								},
								Showtimes: []domain.Showtime{
									{
										ID:        1,
										StartTime: futureTime,
										BasePrice: pgtype.Numeric{
											Int:   big.NewInt(50),
											Exp:   0,
											Valid: true,
										},
									},
									{
										ID:        2,
										StartTime: pastTime,
										BasePrice: pgtype.Numeric{
											Int:   big.NewInt(50),
											Exp:   0,
											Valid: true,
										},
									},
								},
							},
						},
					},
				}
				metadata := &domain.Metadata{
					CurrentPage:  1,
					FirstPage:    1,
					LastPage:     1,
					PageSize:     10,
					TotalRecords: 1,
				}
				return theaters, metadata, nil
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.MovieShowtimesResponse{
				Date: types.Date{Time: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)},
				Theaters: []api.TheaterShowtimes{
					{
						Id:       1,
						Name:     "Test Theater",
						Address:  "123 Test St",
						City:     "Test City",
						District: "Test District",
						Distance: 2.5,
						Amenities: []api.Amenity{
							{Id: 1, Name: "IMAX", Description: "IMAX Screen"},
						},
						Halls: []api.Hall{
							{
								Id:   1,
								Name: "Hall 1",
								Amenities: []api.Amenity{
									{Id: 2, Name: "Dolby", Description: "Dolby Sound"},
								},
								Showtimes: []api.Showtime{
									{
										Id:            1,
										StartDateTime: futureTime,
										StartTime:     futureTime.Format("15:04"),
										Price:         50,
										Status:        api.AVAILABLE,
									},
									{
										Id:            2,
										StartDateTime: pastTime,
										StartTime:     pastTime.Format("15:04"),
										Price:         50,
										Status:        api.EXPIRED,
									},
								},
							},
						},
					},
				},
				Metadata: &api.Metadata{
					CurrentPage:  1,
					FirstPage:    1,
					LastPage:     1,
					PageSize:     10,
					TotalRecords: 1,
				},
			},
		},
		{
			name: "invalid movie ID",
			id:   0,
			params: api.GetMovieShowtimesParams{
				Date:      "2024-03-20",
				Latitude:  39.990067,
				Longitude: 32.643482,
			},
			url:            "/movies/0/showtimes",
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "movie ID must be greater than zero",
		},
		{
			name: "invalid date format",
			id:   1,
			params: api.GetMovieShowtimesParams{
				Date:      "invalid-date",
				Latitude:  39.990067,
				Longitude: 32.643482,
			},
			url:            "/movies/1/showtimes?date=invalid-date",
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: validator.ErrDefaultInvalid,
		},
		{
			name: "database error",
			id:   1,
			params: api.GetMovieShowtimesParams{
				Date:      "2024-03-20",
				Latitude:  39.990067,
				Longitude: 32.643482,
			},
			url: "/movies/1/showtimes",
			getTheatersFunc: func(ctx context.Context, movieID int, date time.Time, lon, lat float64, pagination domain.Pagination) (
				[]domain.Theater,
				*domain.Metadata,
				error,
			) {
				return nil, nil, fmt.Errorf("database error")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name: "empty result",
			id:   1,
			params: api.GetMovieShowtimesParams{
				Date:      "2024-03-20",
				Latitude:  39.990067,
				Longitude: 32.643482,
			},
			url: "/movies/1/showtimes",
			getTheatersFunc: func(ctx context.Context, movieID int, date time.Time, lon, lat float64, pagination domain.Pagination) (
				[]domain.Theater,
				*domain.Metadata,
				error,
			) {
				return []domain.Theater{}, &domain.Metadata{
					CurrentPage:  1,
					FirstPage:    1,
					LastPage:     1,
					PageSize:     10,
					TotalRecords: 0,
				}, nil
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.MovieShowtimesResponse{
				Date:     types.Date{Time: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)},
				Theaters: []api.TheaterShowtimes{},
				Metadata: &api.Metadata{
					CurrentPage:  1,
					FirstPage:    1,
					LastPage:     1,
					PageSize:     10,
					TotalRecords: 0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(func(a *application) {
				a.theaterRepo = &mocks.MockTheaterRepo{
					GetTheatersByMovieAndLocationAndDateFunc: tt.getTheatersFunc,
				}
			})

			w, r := executeRequest(t, http.MethodGet, tt.url, nil)

			app.GetMovieShowtimes(w, r, tt.id, tt.params)

			if got := w.Code; got != tt.wantStatus {
				t.Errorf("GetMovieShowtimes() status = %v, want %v", got, tt.wantStatus)
			}

			if tt.wantResponse != nil {
				var response api.MovieShowtimesResponse

				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if diff := cmp.Diff(tt.wantResponse, &response); diff != "" {
					t.Errorf("GetMovieShowtimes() response mismatch (-want +got):\n%s", diff)
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
