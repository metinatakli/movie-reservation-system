package integration_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/metinatakli/movie-reservation-system/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

const (
	dbName         = "movie_reservation"
	dbUser         = "test_user"
	dbPassword     = "test_password"
	dbImageName    = "postgres:17-alpine"
	cacheImageName = "redis:7"
)

type BaseSuite struct {
	suite.Suite
	app            *TestApp
	dbContainer    *PostgresContainer
	cacheContainer *RedisContainer
	server         *httptest.Server
}

func (s *BaseSuite) SetupSuite() {
	ctx := context.Background()

	postgresContainer, err := getDbContainer(ctx)
	if err != nil {
		log.Printf("failed to start container: %s", err)
		return
	}

	redisContainer, err := getCacheContainer(ctx)
	if err != nil {
		log.Printf("failed to start container: %s", err)
		return
	}

	s.dbContainer = postgresContainer
	s.cacheContainer = redisContainer

	cfg := app.Config{
		Port: 3000,
		Env:  "test",
		DB: app.DBConfig{
			DSN:          postgresContainer.ConnectionString,
			MaxOpenConns: 25,
			MaxIdleTime:  2 * time.Minute,
		},
		Redis: app.RedisConfig{
			URL:          redisContainer.ConnectionString,
			MaxOpenConns: 10,
			MaxIdleConns: 10,
			MaxIdleTime:  2 * time.Minute,
		},
	}

	testApp, err := newTestApp(cfg)
	if err != nil {
		log.Printf("cannot initialize app: %s", err)
		return
	}

	s.app = testApp
	s.server = httptest.NewServer(testApp.App.Routes())
}

func (s *BaseSuite) TearDownSuite() {
	s.server.Close()
	if err := testcontainers.TerminateContainer(s.dbContainer.Container.Container); err != nil {
		log.Printf("failed to terminate container: %s", err)
	}
	if err := testcontainers.TerminateContainer(s.cacheContainer.Container); err != nil {
		log.Printf("failed to terminate container: %s", err)
	}
}

type Scenario struct {
	Name             string
	Method           string
	URL              string
	Body             io.Reader
	Headers          map[string]string
	Cookies          []http.Cookie
	ExpectedStatus   int
	ExpectedResponse string
	BeforeTestFunc   func(t testing.TB, app *TestApp)
	AfterTestFunc    func(t testing.TB, app *TestApp, res *http.Response)
}

func (s Scenario) Run(t *testing.T, testApp *TestApp) {
	t.Run(s.Name, func(t *testing.T) {
		req, err := prepareRequest(s.Method, s.URL, s.Body, s.Headers, s.Cookies)
		require.NoError(t, err)

		if s.BeforeTestFunc != nil {
			s.BeforeTestFunc(t, testApp)
		}

		rec := httptest.NewRecorder()
		testApp.App.Routes().ServeHTTP(rec, req)

		res := rec.Result()
		defer res.Body.Close()

		assert.Equal(t, s.ExpectedStatus, res.StatusCode)

		if s.ExpectedResponse != "" {
			compareResponse(t, res.Body, s.ExpectedResponse)
		}

		if s.AfterTestFunc != nil {
			s.AfterTestFunc(t, testApp, res)
		}
	})
}
