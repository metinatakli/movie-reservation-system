package app

import (
	"net/http"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
)

func (app *application) GetReservationsOfUserHandler(
	w http.ResponseWriter,
	r *http.Request,
	params api.GetReservationsOfUserHandlerParams) {

	err := app.validator.Struct(params)
	if err != nil {
		app.failedValidationResponse(w, r, err)
		return
	}

	userId := app.contextGetUserId(r)
	pagination := toPagination(params)

	reservations, metadata, err := app.reservationRepo.GetReservationsSummariesByUserId(r.Context(), userId, pagination)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	reservationSummaries := toReservationSummaries(reservations)
	apiMetadata := toApiMetadata(metadata)
	resp := api.UserReservationsResponse{
		Reservations: reservationSummaries,
		Metadata:     *apiMetadata,
	}

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func toReservationSummaries(reservations []domain.ReservationSummary) []api.ReservationSummary {
	reservationSummaries := make([]api.ReservationSummary, len(reservations))

	for i, v := range reservations {
		ReservationSummary := &reservationSummaries[i]

		ReservationSummary.Id = v.ReservationID
		ReservationSummary.MovieTitle = v.MovieTitle
		ReservationSummary.MoviePosterUrl = v.MoviePosterUrl
		ReservationSummary.HallName = v.HallName
		ReservationSummary.TheaterName = v.TheaterName
		ReservationSummary.Date = v.ShowtimeDate
		ReservationSummary.CreatedAt = v.CreatedAt
	}

	return reservationSummaries
}

func toPagination(params api.GetReservationsOfUserHandlerParams) domain.Pagination {
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

	return pagination
}
