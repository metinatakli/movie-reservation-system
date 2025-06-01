package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/metinatakli/movie-reservation-system/internal/mocks"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/stripe/stripe-go/v82"
)

type MockUserRepo struct {
	mock.Mock
	domain.UserRepository
}

func (m *MockUserRepo) GetById(ctx context.Context, id int) (*domain.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.User), args.Error(1)
}

type CheckoutSessionTestSuite struct {
	suite.Suite
	app             *Application
	redisClient     *mocks.MockRedisClient
	paymentRepo     *mocks.MockPaymentRepo
	userRepo        *MockUserRepo
	paymentProvider *mocks.MockPaymentProvider
	sessionManager  *scs.SessionManager
}

func (s *CheckoutSessionTestSuite) SetupTest() {
	s.redisClient = new(mocks.MockRedisClient)
	s.paymentRepo = new(mocks.MockPaymentRepo)
	s.userRepo = new(MockUserRepo)
	s.paymentProvider = new(mocks.MockPaymentProvider)
	s.sessionManager = scs.New()

	s.app = newTestApplication(func(a *Application) {
		a.redis = s.redisClient
		a.paymentRepo = s.paymentRepo
		a.userRepo = s.userRepo
		a.sessionManager = s.sessionManager
		a.paymentProvider = s.paymentProvider
	})
}

func TestCheckoutSessionSuite(t *testing.T) {
	suite.Run(t, new(CheckoutSessionTestSuite))
}

func (s *CheckoutSessionTestSuite) TestCreateCheckoutSessionHandler() {
	tests := []struct {
		name           string
		setupMocks     func(string)
		wantStatus     int
		wantErrMessage string
		wantResponse   *api.CheckoutSessionResponse
	}{
		{
			name: "should fail when there is no cart bound to the current session",
			setupMocks: func(sessionId string) {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringResult("", redis.Nil)).Once()
			},
			wantStatus:     http.StatusNotFound,
			wantErrMessage: "there is no cart bound to the current session",
		},
		{
			name: "should fail when fetching cart data fails",
			setupMocks: func(sessionId string) {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringResult("cart-id", nil)).Once()
				s.redisClient.On("Get", mock.Anything, "cart-id").Return(redis.NewStringResult("", fmt.Errorf("redis get operation failed"))).Once()
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name: "should fail when cart data cannot be unmarshalled",
			setupMocks: func(sessionId string) {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringResult("cart-id", nil)).Once()
				s.redisClient.On("Get", mock.Anything, "cart-id").Return(redis.NewStringResult("invalid-cart-data", nil)).Once()
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name: "should fail when the Redis call fails while doing seat ownership check",
			setupMocks: func(sessionId string) {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringResult("cart-id", nil)).Once()
				s.redisClient.On("Get", mock.Anything, "cart-id").Return(redis.NewStringResult(cartDataStr, nil)).Once()
				s.redisClient.On("Get", mock.Anything, mock.Anything).
					Return(redis.NewStringResult("", fmt.Errorf("redis get operation failed"))).Once()
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name: "should fail when seat ownership check fails",
			setupMocks: func(sessionId string) {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringResult("cart-id", nil)).Once()
				s.redisClient.On("Get", mock.Anything, "cart-id").Return(redis.NewStringResult(cartDataStr, nil)).Once()
				s.redisClient.On("Get", mock.Anything, seatLockKey(1, 1)).
					Return(redis.NewStringResult("other-session-id", nil)).Once()
			},
			wantStatus:     http.StatusConflict,
			wantErrMessage: fmt.Sprintf("seat %d doesn't belong to the current session", 1),
		},
		{
			name: "should fail when payment provider fails to create checkout session",
			setupMocks: func(sessionId string) {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringResult("cart-id", nil)).Once()
				s.redisClient.On("Get", mock.Anything, "cart-id").Return(redis.NewStringResult(cartDataStr, nil)).Once()

				// add the mock calls for retrieving seat locks
				s.redisClient.On("Get", mock.Anything, seatLockKey(1, 1)).
					Return(redis.NewStringResult(sessionId, nil)).Once()

				s.redisClient.On("Get", mock.Anything, seatLockKey(1, 2)).
					Return(redis.NewStringResult(sessionId, nil)).Once()

				s.userRepo.On("GetById", mock.Anything, mock.Anything).Return(&domain.User{ID: 1, Email: "test@test.com"}, nil)

				s.paymentProvider.On("CreateCheckoutSession", mock.Anything, mock.Anything, mock.Anything).
					Return(&stripe.CheckoutSession{}, fmt.Errorf("payment provider error"))
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: ErrInternalServer,
		},
		{
			name: "should successfully create checkout session",
			setupMocks: func(sessionId string) {
				s.redisClient.On("Get", mock.Anything, mock.Anything).Return(redis.NewStringResult("cart-id", nil)).Once()
				s.redisClient.On("Get", mock.Anything, "cart-id").Return(redis.NewStringResult(cartDataStr, nil)).Once()

				// add the mock calls for retrieving seat locks
				s.redisClient.On("Get", mock.Anything, seatLockKey(1, 1)).Return(redis.NewStringResult(sessionId, nil)).Once()
				s.redisClient.On("Get", mock.Anything, seatLockKey(1, 2)).Return(redis.NewStringResult(sessionId, nil)).Once()

				s.userRepo.On("GetById", mock.Anything, mock.Anything).
					Return(&domain.User{ID: 1, Email: "test@test.com"}, nil).Once()

				s.paymentProvider.On("CreateCheckoutSession", mock.Anything, mock.Anything, mock.Anything).
					Return(&stripe.CheckoutSession{ID: "checkout-id", URL: "http://payment.url"}, nil)

				s.paymentRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
			},
			wantStatus: http.StatusOK,
			wantResponse: &api.CheckoutSessionResponse{
				RedirectUrl: "http://payment.url",
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()

			defer s.paymentRepo.AssertExpectations(s.T())
			defer s.userRepo.AssertExpectations(s.T())
			defer s.redisClient.AssertExpectations(s.T())
			defer s.paymentProvider.AssertExpectations(s.T())

			w, r := executeRequest(s.T(), http.MethodPost, "/checkout/session", nil)
			r = setupTestSession(s.T(), s.app, r, 1)

			if tt.setupMocks != nil {
				sessionId := s.app.sessionManager.Token(r.Context())
				tt.setupMocks(sessionId)
			}

			handler := http.Handler(http.HandlerFunc(s.app.CreateCheckoutSessionHandler))
			handler = s.app.sessionManager.LoadAndSave(handler)
			handler = s.app.requireAuthentication(handler)
			handler.ServeHTTP(w, r)

			s.Equal(tt.wantStatus, w.Code)

			if tt.wantResponse != nil {
				var response api.CheckoutSessionResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				s.Require().NoError(err, "Failed to decode response")

				s.Equal(tt.wantResponse.RedirectUrl, response.RedirectUrl)
			}

			checkErrorResponse(s.T(), w, struct {
				wantStatus     int
				wantErrMessage string
			}{
				wantStatus:     tt.wantStatus,
				wantErrMessage: tt.wantErrMessage,
			})
		})
	}
}
