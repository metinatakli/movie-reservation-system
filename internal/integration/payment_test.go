package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/stripe/stripe-go/v82"
)

type CheckoutTestSuite struct {
	BaseSuite
}

func TestCheckoutSuite(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	suite.Run(t, new(CheckoutTestSuite))
}

func (s *CheckoutTestSuite) TestCreateCheckoutSessionHandler() {
	cookies := s.app.authenticatedUserCookies(s.T())

	mockStripeSession := &stripe.CheckoutSession{
		ID:  TestCheckoutSessionId,
		URL: TestCheckoutSessionURL,
	}

	scenarios := []Scenario{
		{
			Name:             "returns 401 if an attempt is made without authentication",
			Method:           "POST",
			URL:              "/checkout/session",
			ExpectedStatus:   http.StatusUnauthorized,
			ExpectedResponse: `{"message": "You must be authenticated to access this resource"}`,
		},
		{
			Name:             "returns 404 if no cart exists in the session",
			Method:           "POST",
			URL:              "/checkout/session",
			Cookies:          cookies,
			ExpectedStatus:   http.StatusNotFound,
			ExpectedResponse: `{"message": "there is no cart bound to the current session"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/users_down.sql")
				setupBaseCreateCartHandlerState(t, app)
				executeSQLFile(t, app.DB, "testdata/users_up.sql")
			},
		},
		{
			Name:             "returns 409 if a seat lock has expired",
			Method:           "POST",
			URL:              "/checkout/session",
			Cookies:          cookies,
			ExpectedStatus:   http.StatusConflict,
			ExpectedResponse: `{"message": "your selections have expired, please select your seats again"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/users_down.sql")
				setupBaseCreateCartHandlerState(t, app)
				executeSQLFile(t, app.DB, "testdata/users_up.sql")
				truncateCartRelatedCache(t, app.RedisClient)

				// Create a cart that lives for a minute, but seat locks that expire instantly.
				createTestCartInCache(t, app, cookies[0].Value, 1, []domain.CartSeat{{Id: 1}, {Id: 4}}, 1*time.Minute, 1*time.Millisecond)
				time.Sleep(5 * time.Millisecond)
			},
		},
		{
			Name:             "returns 500 if the authenticated user is not in the database",
			Method:           "POST",
			URL:              "/checkout/session",
			Cookies:          cookies,
			ExpectedStatus:   http.StatusInternalServerError,
			ExpectedResponse: `{"message": "The server encountered a problem and could not process your request"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				setupBaseCreateCartHandlerState(t, app)
				executeSQLFile(t, app.DB, "testdata/users_down.sql")
				truncateCartRelatedCache(t, app.RedisClient)

				createTestCartInCache(t, app, cookies[0].Value, 1, []domain.CartSeat{{Id: 1}}, 10*time.Minute, 10*time.Minute)
			},
		},
		{
			Name:             "returns 500 if the payment provider fails",
			Method:           "POST",
			URL:              "/checkout/session",
			Cookies:          cookies,
			ExpectedStatus:   http.StatusInternalServerError,
			ExpectedResponse: `{"message": "The server encountered a problem and could not process your request"}`,
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/users_down.sql")
				setupBaseCreateCartHandlerState(t, app)
				executeSQLFile(t, app.DB, "testdata/users_up.sql")
				truncateCartRelatedCache(t, app.RedisClient)

				createTestCartInCache(t, app, cookies[0].Value, 1, []domain.CartSeat{{Id: 1}}, 10*time.Minute, 10*time.Minute)
				app.PaymentProvider.Err = errors.New("stripe api is down")
			},
		},
		{
			Name:             "successfully creates a checkout session",
			Method:           "POST",
			URL:              "/checkout/session",
			Cookies:          cookies,
			ExpectedStatus:   http.StatusOK,
			ExpectedResponse: fmt.Sprintf(`{"redirectUrl": "%s"}`, mockStripeSession.URL),
			BeforeTestFunc: func(t testing.TB, app *TestApp) {
				executeSQLFile(t, app.DB, "testdata/users_down.sql")
				setupBaseCreateCartHandlerState(t, app)
				executeSQLFile(t, app.DB, "testdata/users_up.sql")
				truncateCartRelatedCache(t, app.RedisClient)

				createTestCartInCache(
					t,
					app,
					cookies[0].Value,
					1,
					[]domain.CartSeat{
						{
							Id:         1,
							ExtraPrice: decimal.NewFromFloat(0),
						},
						{
							Id:         4,
							ExtraPrice: decimal.NewFromFloat(8.49),
						},
					},
					10*time.Minute,
					10*time.Minute,
				)
				app.PaymentProvider.CheckoutSession = mockStripeSession
			},
			AfterTestFunc: func(t testing.TB, app *TestApp, res *http.Response) {
				var p domain.Payment
				query := `SELECT user_id, amount, status FROM payments ORDER BY created_at DESC LIMIT 1`
				err := app.DB.QueryRow(context.Background(), query).Scan(&p.UserID, &p.Amount, &p.Status)
				require.NoError(t, err)

				require.Equal(t, 1, p.UserID, "expected user ID to be 1")
				require.Equal(t, "8.49", p.Amount.StringFixed(2), "expected amounts of payment and cart to match")
				require.Equal(t, domain.PaymentStatusPending, p.Status, "expected payment status to be pending")
			},
		},
	}

	for _, scenario := range scenarios {
		s.app.PaymentProvider.CheckoutSession = nil
		s.app.PaymentProvider.Err = nil

		scenario.Run(s.T(), s.app)
	}
}
