package integration_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
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

var testApp *TestApp

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Setup containers
	dbContainer, err := getDbContainer(ctx)
	if err != nil {
		log.Printf("failed to spin db container up: %s", err)
		os.Exit(1)
	}

	cacheContainer, err := getCacheContainer(ctx)
	if err != nil {
		log.Printf("failed to spin cache container up: %s", err)
		os.Exit(1)
	}

	// Setup test app
	cfg := app.Config{
		Port: 3000,
		Env:  "test",
		DB: app.DBConfig{
			DSN:          dbContainer.ConnectionString,
			MaxOpenConns: 25,
			MaxIdleTime:  2 * time.Minute,
		},
		Redis: app.RedisConfig{
			URL:          cacheContainer.ConnectionString,
			MaxOpenConns: 10,
			MaxIdleConns: 10,
			MaxIdleTime:  2 * time.Minute,
		},
	}

	testApp, err = newTestApp(cfg)
	if err != nil {
		log.Printf("cannot initialize app: %s", err)
		os.Exit(1)
	}

	// Run all tests
	code := m.Run()

	// Clean up containers after all tests are done
	if dbContainer != nil {
		if err := testcontainers.TerminateContainer(dbContainer.Container.Container); err != nil {
			log.Printf("failed to terminate db container: %s", err)
		}
	}
	if cacheContainer != nil {
		if err := testcontainers.TerminateContainer(cacheContainer.Container); err != nil {
			log.Printf("failed to terminate cache container: %s", err)
		}
	}

	os.Exit(code)
}

type BaseSuite struct {
	suite.Suite
	app    *TestApp
	server *httptest.Server
}

func (s *BaseSuite) SetupSuite() {
	s.app = testApp
	s.server = httptest.NewServer(testApp.App.Routes())
}

func (s *BaseSuite) TearDownSuite() {
	s.server.Close()
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
	t.Helper()

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
