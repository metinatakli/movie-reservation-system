package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
		getAllFunc     func(context.Context, domain.MovieFilters) ([]*domain.Movie, *domain.Metadata, error)
		wantStatus     int
		wantErrMessage string
		wantResponse   *api.MovieListResponse
	}{
		{
			name:   "successful retrieval with default parameters",
			params: api.GetMoviesParams{},
			url:    "/movies",
			getAllFunc: func(ctx context.Context, filters domain.MovieFilters) ([]*domain.Movie, *domain.Metadata, error) {
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
			getAllFunc: func(ctx context.Context, filters domain.MovieFilters) ([]*domain.Movie, *domain.Metadata, error) {
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
			wantErrMessage: fmt.Sprintf(validator.ErrMinLength, "1"),
		},
		{
			name: "validation error - page size too large",
			params: api.GetMoviesParams{
				PageSize: ptr(1000),
			},
			url:            "/movies?pageSize=1000",
			wantStatus:     http.StatusUnprocessableEntity,
			wantErrMessage: fmt.Sprintf(validator.ErrMaxLength, "100"),
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
			getAllFunc: func(ctx context.Context, filters domain.MovieFilters) ([]*domain.Movie, *domain.Metadata, error) {
				return nil, nil, fmt.Errorf("database connection error")
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name:   "empty result",
			params: api.GetMoviesParams{},
			url:    "/movies",
			getAllFunc: func(ctx context.Context, filters domain.MovieFilters) ([]*domain.Movie, *domain.Metadata, error) {
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
