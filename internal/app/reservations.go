package app

import (
	"errors"
	"fmt"
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

func (app *application) GetUserReservationById(w http.ResponseWriter, r *http.Request, reservationId int) {
	if reservationId <= 0 {
		app.badRequestResponse(w, r, fmt.Errorf("reservation id must be greater than zero"))
		return
	}

	userId := app.contextGetUserId(r)

	reservationDetail, err := app.reservationRepo.GetByReservationIdAndUserId(r.Context(), reservationId, userId)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	resp := toReservationDetailResponse(reservationDetail)

	err = app.writeJSON(w, http.StatusOK, resp, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func toReservationDetailResponse(reservationDetail *domain.ReservationDetail) api.ReservationDetailResponse {
	seats := make([]api.ReservationSeat, len(reservationDetail.Seats))
	for i, s := range reservationDetail.Seats {
		seats[i] = api.ReservationSeat{
			Row:    s.Row,
			Column: s.Col,
			Type:   api.SeatType(s.Type),
		}
	}

	theaterAmenities := make([]api.Amenity, len(reservationDetail.TheaterAmenities))
	for i, a := range reservationDetail.TheaterAmenities {
		theaterAmenities[i] = api.Amenity{
			Id:          a.ID,
			Name:        a.Name,
			Description: a.Description,
		}
	}

	hallAmenities := make([]api.Amenity, len(reservationDetail.HallAmenities))
	for i, a := range reservationDetail.HallAmenities {
		hallAmenities[i] = api.Amenity{
			Id:          a.ID,
			Name:        a.Name,
			Description: a.Description,
		}
	}

	return api.ReservationDetailResponse{
		Id:               reservationDetail.ReservationID,
		MovieTitle:       reservationDetail.MovieTitle,
		MoviePosterUrl:   reservationDetail.MoviePosterUrl,
		Date:             reservationDetail.ShowtimeDate,
		TheaterName:      reservationDetail.TheaterName,
		HallName:         reservationDetail.HallName,
		CreatedAt:        reservationDetail.CreatedAt,
		Seats:            seats,
		TheaterAmenities: &theaterAmenities,
		HallAmenities:    &hallAmenities,
		TotalPrice:       reservationDetail.TotalPrice,
	}
}
