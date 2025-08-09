package payment

import (
	"fmt"
	"strconv"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)

type StripePaymentProvider struct {
	failureUrl string
	successUrl string
}

func NewStripePaymentProvider(failureUrl, successUrl string) *StripePaymentProvider {
	return &StripePaymentProvider{
		failureUrl: failureUrl,
		successUrl: successUrl,
	}
}

func (s *StripePaymentProvider) CreateCheckoutSession(
	sessionId string,
	user *domain.User,
	cart domain.Cart,
	payment domain.Payment) (*stripe.CheckoutSession, error) {

	var lineItems []*stripe.CheckoutSessionLineItemParams

	for _, seat := range cart.Seats {
		seatLabel := fmt.Sprintf("Row %d Seat %d", seat.Row, seat.Col)

		seatPrice := cart.BasePrice.Add(seat.ExtraPrice)
		priceCents := seatPrice.Mul(decimal.NewFromInt(100)).IntPart()

		lineItem := &stripe.CheckoutSessionLineItemParams{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String(string(stripe.CurrencyUSD)),
				UnitAmount: stripe.Int64(priceCents),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String(fmt.Sprintf("🎬 %s - %s", cart.MovieName, seatLabel)),
					Description: stripe.String(fmt.Sprintf(
						"Theater: %s • Hall: %s • Showtime: %s • Seat Type: %s",
						cart.TheaterName,
						cart.HallName,
						cart.Date.Format("Jan 2, 2006 15:04"),
						seat.SeatType,
					)),
				},
			},
			Quantity: stripe.Int64(1),
		}

		lineItems = append(lineItems, lineItem)
	}

	params := &stripe.CheckoutSessionParams{
		LineItems:  lineItems,
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(s.successUrl),
		CancelURL:  stripe.String(s.failureUrl),
		Metadata: map[string]string{
			"cart_id":    cart.Id,
			"session_id": sessionId,
			"user_id":    strconv.Itoa(user.ID),
			"payment_id": strconv.Itoa(payment.ID),
		},
		CustomerEmail:     &user.Email,
		ClientReferenceID: stripe.String(strconv.Itoa(user.ID)),
	}

	return session.New(params)
}
