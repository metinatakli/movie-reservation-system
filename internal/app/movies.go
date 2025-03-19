package app

import (
	"errors"
	"fmt"
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

func toMovieFilters(params api.GetMoviesParams) domain.Pagination {
	filters := domain.Pagination{
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

func (app *application) ShowMovieDetails(w http.ResponseWriter, r *http.Request, id int) {
	if id < 1 {
		app.badRequestResponse(w, r, fmt.Errorf("movie ID must be greater than zero"))
		return
	}

	movie, err := app.movieRepo.GetById(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	resp := toMovieDetailsResponse(movie)

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func toMovieDetailsResponse(movie *domain.Movie) api.MovieDetailsResponse {
	if movie == nil {
		return api.MovieDetailsResponse{}
	}

	resp := api.MovieDetailsResponse{
		Id:          movie.ID,
		Name:        movie.Title,
		PosterUrl:   movie.PosterUrl,
		ReleaseDate: types.Date{Time: movie.ReleaseDate},
		Description: movie.Description,
		Runtime:     movie.Duration,
		Genres:      movie.Genres,
		Language:    movie.Language,
		Director:    movie.Director,
		Cast:        movie.CastMembers,
	}

	if movie.Rating.Valid {
		float64Value, floatErr := movie.Rating.Float64Value()
		if floatErr == nil {
			val := float32(float64Value.Float64)
			resp.Rating = &val
		}
	}

	return resp
}

func (app *application) GetMovieShowtimes(
	w http.ResponseWriter,
	r *http.Request,
	id int,
	params api.GetMovieShowtimesParams) {

	if id < 1 {
		app.badRequestResponse(w, r, fmt.Errorf("movie ID must be greater than zero"))
		return
	}

	err := app.validator.Struct(params)
	if err != nil {
		app.failedValidationResponse(w, r, err)
		return
	}

	date, err := time.Parse(time.DateOnly, params.Date)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	pagination := domain.Pagination{
		Page:     DefaultPage,
		PageSize: DefaultPageSize,
	}

	if params.Page != nil {
		pagination.Page = *params.Page
	}

	if params.PageSize != nil {
		pagination.PageSize = *params.PageSize
	}

	theaters, metadata, err := app.theaterRepo.GetTheatersByMovieAndLocationAndDate(
		r.Context(),
		id,
		date,
		params.Longitude,
		params.Latitude,
		pagination,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	theaterShowtimes := toTheaterShowtimes(theaters)
	apiMetadata := toApiMetadata(metadata)

	resp := api.MovieShowtimesResponse{
		Date:     types.Date{Time: date},
		Theaters: theaterShowtimes,
		Metadata: apiMetadata,
	}

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func toTheaterShowtimes(theaters []domain.Theater) []api.TheaterShowtimes {
	theaterShowtimes := make([]api.TheaterShowtimes, len(theaters))

	for i, v := range theaters {
		theaterShowtime := toTheaterShowtime(v)
		theaterShowtimes[i] = theaterShowtime
	}

	return theaterShowtimes
}

func toTheaterShowtime(theater domain.Theater) api.TheaterShowtimes {
	return api.TheaterShowtimes{
		Address:   theater.Address,
		Amenities: toAmenities(theater.Amenities),
		City:      theater.City,
		Distance:  theater.Distance,
		District:  theater.District,
		Halls:     toHalls(theater.Halls),
		Id:        theater.ID,
		Name:      theater.Name,
	}
}

func toAmenities(amenities []domain.Amenity) []api.Amenity {
	apiAmenities := make([]api.Amenity, len(amenities))

	for i, v := range amenities {
		amenity := api.Amenity{
			Id:          v.ID,
			Name:        v.Name,
			Description: v.Description,
		}

		apiAmenities[i] = amenity
	}

	return apiAmenities
}

func toHalls(halls []domain.Hall) []api.Hall {
	apiHalls := make([]api.Hall, len(halls))

	for i, v := range halls {
		hall := api.Hall{
			Id:        v.ID,
			Amenities: toAmenities(v.Amenities),
			Name:      v.Name,
			Showtimes: toShowtimes(v.Showtimes),
		}

		apiHalls[i] = hall
	}

	return apiHalls
}

func toShowtimes(showtimes []domain.Showtime) []api.Showtime {
	apiShowtimes := make([]api.Showtime, len(showtimes))
	now := time.Now()

	for i, v := range showtimes {
		showtime := api.Showtime{
			Id:            v.ID,
			StartDateTime: v.StartTime,
			StartTime:     v.StartTime.Format("15:04"),
		}

		if v.BasePrice.Valid {
			float64Value, floatErr := v.BasePrice.Float64Value()
			if floatErr == nil {
				showtime.Price = float32(float64Value.Float64)
			}
		}

		// TODO: Add SOLD_OUT
		if showtime.StartDateTime.Before(now) {
			showtime.Status = api.EXPIRED
		} else {
			showtime.Status = api.AVAILABLE
		}

		apiShowtimes[i] = showtime
	}

	return apiShowtimes
}
