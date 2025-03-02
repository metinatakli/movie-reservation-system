package app

import (
	"net/http"
	"time"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/oapi-codegen/runtime/types"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 10
	DefaultSort     = "id"
)

func (app *application) GetMovies(w http.ResponseWriter, r *http.Request, params api.GetMoviesParams) {
	err := app.validator.Struct(params)
	if err != nil {
		app.failedValidationResponse(w, r, err)
		return
	}

	filters := toMovieFilters(params)

	movies, metadata, err := app.movieRepo.GetAll(r.Context(), filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	movieSummaries := toMovieSummaries(movies)
	apiMetadata := toApiMetadata(metadata)

	resp := api.MovieListResponse{
		Movies:   movieSummaries,
		Metadata: apiMetadata,
	}

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func toMovieFilters(params api.GetMoviesParams) domain.MovieFilters {
	filters := domain.MovieFilters{
		Page:     DefaultPage,
		PageSize: DefaultPageSize,
		Sort:     DefaultSort,
	}

	if params.Page != nil {
		filters.Page = *params.Page
	}
	if params.PageSize != nil {
		filters.PageSize = *params.PageSize
	}
	if params.Sort != nil {
		filters.Sort = *params.Sort
	}
	if params.Term != nil {
		filters.Term = *params.Term
	}

	return filters
}

func toMovieSummaries(movies []*domain.Movie) []api.MovieSummary {
	summaries := make([]api.MovieSummary, len(movies))
	today := time.Now().Truncate(24 * time.Hour)

	for i, movie := range movies {
		summary := toMovieSummary(movie)

		if movie.ReleaseDate.After(today) {
			summary.Status = api.COMINGSOON
		} else {
			summary.Status = api.NOWSHOWING
		}

		summaries[i] = summary
	}

	return summaries
}

func toMovieSummary(movie *domain.Movie) api.MovieSummary {
	if movie == nil {
		return api.MovieSummary{}
	}

	return api.MovieSummary{
		Id:          movie.ID,
		Name:        movie.Title,
		Description: movie.Description,
		PosterUrl:   movie.PosterUrl,
		ReleaseDate: types.Date{Time: movie.ReleaseDate},
	}
}

func toApiMetadata(metadata *domain.Metadata) *api.Metadata {
	if metadata == nil {
		return nil
	}

	return &api.Metadata{
		CurrentPage:  metadata.CurrentPage,
		FirstPage:    metadata.FirstPage,
		LastPage:     metadata.LastPage,
		PageSize:     metadata.PageSize,
		TotalRecords: metadata.TotalRecords,
	}
}
