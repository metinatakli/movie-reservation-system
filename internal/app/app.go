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
	"github.com/exaring/otelpgx"
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
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/riandyrn/otelchi"
	"github.com/stripe/stripe-go/v82"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/log/global"
)

var (
	version = vcs.Version()
)

type Application struct {
	config         Config
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

type DBConfig struct {
	DSN          string
	MaxOpenConns int
	MaxIdleTime  time.Duration
}

type RedisConfig struct {
	URL          string
	MaxOpenConns int
	MaxIdleConns int
	MaxIdleTime  time.Duration
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Sender   string
}

type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	SuccessURL    string
	FailureURL    string
}

type Config struct {
	Port             int
	Env              string
	DB               DBConfig
	Redis            RedisConfig
	SMTP             SMTPConfig
	Stripe           StripeConfig
	OtelCollectorUrl string
}

func loadFlags() Config {
	var cfg Config

	flag.IntVar(&cfg.Port, "port", 3000, "server port")
	flag.StringVar(&cfg.Env, "env", "dev", "Environment (dev|staging|prod)")

	flag.StringVar(&cfg.DB.DSN, "db-dsn", "", "PostgreSQL DSN")
	flag.IntVar(&cfg.DB.MaxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.DurationVar(&cfg.DB.MaxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max idle time for connections")

	flag.StringVar(&cfg.Redis.URL, "redis-url", "", "Redis URL")
	flag.IntVar(&cfg.Redis.MaxOpenConns, "redis-max-open-conns", 25, "Redis max open connections")
	flag.IntVar(&cfg.Redis.MaxIdleConns, "redis-max-idle-conns", 10, "Redis max idle connections")
	flag.DurationVar(&cfg.Redis.MaxIdleTime, "redis-max-idle-time", 2*time.Minute, "Redis max idle time for connections")

	flag.StringVar(&cfg.SMTP.Host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.SMTP.Port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.SMTP.Username, "smtp-username", "", "SMTP username")
	flag.StringVar(&cfg.SMTP.Password, "smtp-password", "", "SMTP password")
	flag.StringVar(&cfg.SMTP.Sender, "smtp-sender", "CineX <no-reply@cinex.metinatakli.net>", "SMTP sender")

	flag.StringVar(&cfg.Stripe.SecretKey, "stripe-key", "", "Stripe secret key")
	flag.StringVar(&cfg.Stripe.WebhookSecret, "stripe-webhook-secret", "", "Stripe webhook secret")
	flag.StringVar(&cfg.Stripe.SuccessURL, "stripe-success-url", "https://example.com/success.html", "Stripe payment success page")
	flag.StringVar(&cfg.Stripe.FailureURL, "stripe-failure-url", "https://example.com/failure.html", "Stripe payment failure page")

	flag.StringVar(&cfg.OtelCollectorUrl, "otel-collector-url", "", "OpenTelemetry collector URL")

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

	return cfg
}

func newApp(cfg Config, logHandler slog.Handler) (*Application, error) {
	stripe.Key = cfg.Stripe.SecretKey

	logger := slog.New(logHandler)

	validator := appvalidator.NewValidator()

	mailer := mailer.NewSMTPMailer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.Sender)

	db, err := NewDatabasePool(cfg)
	if err != nil {
		return nil, err
	}

	redisClient, err := NewRedisClient(cfg)
	if err != nil {
		db.Close()
		return nil, err
	}

	sessionManager := NewSessionManager(redisClient)

	userRepo := repository.NewPostgresUserRepository(db)
	tokenRepo := repository.NewPostgresTokenRepository(db)
	movieRepo := repository.NewPostgresMovieRepository(db)
	theaterRepo := repository.NewPostgresTheaterRepository(db)
	seatRepo := repository.NewPostgresSeatRepository(db)
	paymentRepo := repository.NewPostgresPaymentRepository(db)
	reservationRepo := repository.NewPostgresReservationRepository(db)

	stripeProvider := payment.NewStripePaymentProvider(cfg.Stripe.FailureURL, cfg.Stripe.SuccessURL)

	app := NewApp(
		cfg,
		logger,
		db,
		redisClient,
		validator,
		mailer,
		sessionManager,
		userRepo,
		tokenRepo,
		movieRepo,
		theaterRepo,
		seatRepo,
		paymentRepo,
		reservationRepo,
		stripeProvider,
	)

	return app, nil
}

func NewApp(
	cfg Config,
	logger *slog.Logger,
	db *pgxpool.Pool,
	redisClient redis.UniversalClient,
	validator *validator.Validate,
	mailer mailer.Mailer,
	sessionManager *scs.SessionManager,
	userRepo domain.UserRepository,
	tokenRepo domain.TokenRepository,
	movieRepo domain.MovieRepository,
	theaterRepo domain.TheaterRepository,
	seatRepo domain.SeatRepository,
	paymentRepo domain.PaymentRepository,
	reservationRepo domain.ReservationRepository,
	paymentProvider domain.PaymentProvider,
) *Application {

	return &Application{
		config:          cfg,
		logger:          logger,
		db:              db,
		redis:           redisClient,
		validator:       validator,
		mailer:          mailer,
		sessionManager:  sessionManager,
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		movieRepo:       movieRepo,
		theaterRepo:     theaterRepo,
		seatRepo:        seatRepo,
		paymentRepo:     paymentRepo,
		reservationRepo: reservationRepo,
		paymentProvider: paymentProvider,
	}
}

func Run() error {
	cfg := loadFlags()

	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)

	app, err := newApp(cfg, jsonHandler)
	if err != nil {
		return err
	}

	otelShutdown, err := app.InitTelemetry()
	if err != nil {
		app.logger.Error("failed to initialize telemetry", "error", err)
		return err
	}

	var finalHandler slog.Handler
	loggerProvider := global.GetLoggerProvider()

	if loggerProvider != nil {
		otelHandler := otelslog.NewHandler("movie-reservation-api", otelslog.WithLoggerProvider(loggerProvider))

		finalHandler = NewMultiHandler(jsonHandler, otelHandler)
		app.logger = slog.New(finalHandler)
	}

	defer func() {
		app.logger.Info("shutting down OpenTelemetry")
		otelShutdown(context.Background())
	}()

	defer app.db.Close()
	defer app.redis.Close()

	return app.run()
}

func NewSessionManager(client *redis.Client) *scs.SessionManager {
	sessionManager := scs.New()

	sessionManager.Store = goredisstore.New(client)
	sessionManager.IdleTimeout = 20 * time.Minute
	sessionManager.Cookie.Name = "session_id"

	return sessionManager
}

func NewRedisClient(cfg Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:            cfg.Redis.URL,
		MaxIdleConns:    cfg.Redis.MaxIdleConns,
		MaxActiveConns:  cfg.Redis.MaxOpenConns,
		ConnMaxIdleTime: cfg.Redis.MaxIdleTime,
	})

	if err := redisotel.InstrumentTracing(rdb); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := rdb.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return rdb, nil
}

func NewDatabasePool(cfg Config) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(cfg.DB.DSN)
	if err != nil {
		return nil, err
	}

	config.MaxConnIdleTime = cfg.DB.MaxIdleTime
	config.MaxConns = int32(cfg.DB.MaxOpenConns)
	config.ConnConfig.Tracer = otelpgx.NewTracer()

	db, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	if err := otelpgx.RecordStats(db); err != nil {
		return nil, fmt.Errorf("unable to record database stats: %w", err)
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

func (app *Application) run() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", app.config.Port),
		Handler:      app.Routes(),
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

	app.logger.Info("starting server", "addr", srv.Addr, "env", app.config.Env)

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

func (app *Application) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(app.recoverPanic)
	r.Use(otelchi.Middleware("movie-reservation-api", otelchi.WithChiRoutes(r)))
	r.Use(app.sessionManager.LoadAndSave)
	r.Use(app.ensureGuestUserSession)
	r.Use(app.loggingMiddleware)

	h := api.HandlerFromMux(app, chi.NewRouter())

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

	r.NotFound(app.notFoundResponse)

	return r
}
