package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/alexedwards/scs/goredisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/metinatakli/movie-reservation-system/internal/mailer"
	"github.com/metinatakli/movie-reservation-system/internal/payment"
	"github.com/metinatakli/movie-reservation-system/internal/repository"
	appvalidator "github.com/metinatakli/movie-reservation-system/internal/validator"
	"github.com/metinatakli/movie-reservation-system/internal/vcs"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v82"
)

var (
	version = vcs.Version()
)

type application struct {
	config         config
	logger         *slog.Logger
	db             *pgxpool.Pool
	redis          redis.UniversalClient
	validator      *validator.Validate
	mailer         mailer.Mailer
	sessionManager *scs.SessionManager

	userRepo        domain.UserRepository
	tokenRepo       domain.TokenRepository
	movieRepo       domain.MovieRepository
	theaterRepo     domain.TheaterRepository
	seatRepo        domain.SeatRepository
	paymentRepo     domain.PaymentRepository
	reservationRepo domain.ReservationRepository

	paymentProvider domain.PaymentProvider
}

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleTime  time.Duration
	}
	redis struct {
		url          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	stripe struct {
		secretKey     string
		webhookSecret string
		successUrl    string
		failureUrl    string
	}
}

func Run() error {
	var cfg config

	flag.IntVar(&cfg.port, "port", 3000, "server port")
	flag.StringVar(&cfg.env, "env", "dev", "Environment (dev|staging|prod)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", "", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max idle time for connections")

	flag.StringVar(&cfg.redis.url, "redis-url", "", "Redis URL")
	flag.IntVar(&cfg.redis.maxOpenConns, "redis-max-open-conns", 25, "Redis max open connections")
	flag.IntVar(&cfg.redis.maxIdleConns, "redis-max-idle-conns", 10, "Redis max idle connections")
	flag.DurationVar(&cfg.redis.maxIdleTime, "redis-max-idle-time", 2*time.Minute, "Redis max idle time for connections")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "CineX <no-reply@cinex.metinatakli.net>", "SMTP sender")

	flag.StringVar(&cfg.stripe.secretKey, "stripe-key", "", "Stripe secret key")
	flag.StringVar(&cfg.stripe.webhookSecret, "stripe-webhook-secret", "", "Stripe webhook secret")
	flag.StringVar(&cfg.stripe.successUrl, "stripe-success-url", "https://example.com/success.html", "Stripe payment success page")
	flag.StringVar(&cfg.stripe.failureUrl, "stripe-failure-url", "https://example.com/failure.html", "Stripe payment failure page")

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

	stripe.Key = cfg.stripe.secretKey

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	validator := appvalidator.NewValidator()

	db, err := newDatabasePool(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	userRepo := repository.NewPostgresUserRepository(db)
	tokenRepo := repository.NewPostgresTokenRepository(db)
	movieRepo := repository.NewPostgresMovieRepository(db)
	theaterRepo := repository.NewPostgresTheaterRepository(db)
	seatRepo := repository.NewPostgresSeatRepository(db)
	paymentRepo := repository.NewPostgresPaymentRepository(db)
	reservationRepo := repository.NewPostgresReservationRepository(db)

	stripeProvider := payment.NewStripePaymentProvider(cfg.stripe.failureUrl, cfg.stripe.successUrl)

	redisClient, err := newRedisClient(cfg)
	if err != nil {
		return err
	}
	defer redisClient.Close()

	app := &application{
		config:          cfg,
		logger:          logger,
		db:              db,
		redis:           redisClient,
		validator:       validator,
		mailer:          mailer.NewSMTPMailer(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
		sessionManager:  newSessionManager(redisClient),
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		movieRepo:       movieRepo,
		theaterRepo:     theaterRepo,
		seatRepo:        seatRepo,
		paymentRepo:     paymentRepo,
		reservationRepo: reservationRepo,
		paymentProvider: stripeProvider,
	}

	return app.run()
}

func newSessionManager(client *redis.Client) *scs.SessionManager {
	sessionManager := scs.New()

	sessionManager.Store = goredisstore.New(client)
	sessionManager.IdleTimeout = 20 * time.Minute
	sessionManager.Cookie.Name = "session_id"

	return sessionManager
}

func newRedisClient(cfg config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:            cfg.redis.url,
		MaxIdleConns:    cfg.redis.maxIdleConns,
		MaxActiveConns:  cfg.redis.maxOpenConns,
		ConnMaxIdleTime: cfg.redis.maxIdleTime,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := rdb.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return rdb, nil
}

func newDatabasePool(cfg config) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	config.MaxConnIdleTime = cfg.db.maxIdleTime
	config.MaxConns = int32(cfg.db.maxOpenConns)

	db, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = db.Ping(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func (app *application) run() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelDebug),
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.logger.Info("shutting down server", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		shutdownError <- nil
	}()

	app.logger.Info("starting server", "addr", srv.Addr, "env", app.config.env)

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	app.logger.Info("stopped server", "addr", srv.Addr)

	return nil
}

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.NotFound(app.notFoundResponse)

	r.Use(middleware.Logger)
	r.Use(middleware.RequestID)
	r.Use(app.recoverPanic)
	r.Use(app.sessionManager.LoadAndSave)
	r.Use(app.ensureGuestUserSession)

	h := api.HandlerFromMux(app, r)

	r.Mount("/", h)

	r.With(app.requireAuthentication).Route("/users/me", func(r chi.Router) {
		r.Get("/", app.GetCurrentUser)
		r.Patch("/", app.UpdateUser)
	})

	r.With(app.requireAuthentication).Route("/users/me/deletion-request", func(r chi.Router) {
		r.Post("/", app.InitiateUserDeletion)
		r.Put("/", app.CompleteUserDeletion)
	})

	r.With(app.requireAuthentication).Route("/users/me/reservations", func(r chi.Router) {
		r.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			params := api.GetReservationsOfUserHandlerParams{}

			if page := r.URL.Query().Get("page"); page != "" {
				if pageNum, err := strconv.Atoi(page); err == nil {
					params.Page = &pageNum
				}
			}

			if pageSize := r.URL.Query().Get("pageSize"); pageSize != "" {
				if pageSizeNum, err := strconv.Atoi(pageSize); err == nil {
					params.PageSize = &pageSizeNum
				}
			}
			app.GetReservationsOfUserHandler(w, r, params)
		}))
	})

	// TODO: Search for a better way to handle these middlewares
	r.With(app.requireAuthentication).Route("/users/me/reservations/{reservationId}", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			reservationIdStr := chi.URLParam(r, "reservationId")
			reservationId, err := strconv.Atoi(reservationIdStr)
			if err != nil {
				app.badRequestResponse(w, r, fmt.Errorf("invalid reservation ID"))
				return
			}
			app.GetUserReservationById(w, r, reservationId)
		})
	})

	r.With(app.requireAuthentication).Route("/checkout/session", func(r chi.Router) {
		r.Post("/", app.CreateCheckoutSessionHandler)
	})

	r.Route("/webhook", func(r chi.Router) {
		r.Post("/", app.StripeWebhookHandler)
	})

	return r
}
