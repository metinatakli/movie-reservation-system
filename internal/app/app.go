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
	"syscall"
	"time"

	"github.com/alexedwards/scs/redisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/domain"
	"github.com/metinatakli/movie-reservation-system/internal/mailer"
	"github.com/metinatakli/movie-reservation-system/internal/repository"
	appvalidator "github.com/metinatakli/movie-reservation-system/internal/validator"
	"github.com/metinatakli/movie-reservation-system/internal/vcs"
)

var (
	version = vcs.Version()
)

type application struct {
	config         config
	logger         *slog.Logger
	db             *pgxpool.Pool
	redis          *redis.Pool
	validator      *validator.Validate
	mailer         mailer.Mailer
	sessionManager *scs.SessionManager

	userRepo    domain.UserRepository
	tokenRepo   domain.TokenRepository
	movieRepo   domain.MovieRepository
	theaterRepo domain.TheaterRepository
	seatRepo    domain.SeatRepository
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

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

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

	redis, err := newRedisPool(cfg)
	if err != nil {
		return err
	}
	defer redis.Close()

	app := &application{
		config:         cfg,
		logger:         logger,
		db:             db,
		redis:          redis,
		validator:      validator,
		mailer:         mailer.NewSMTPMailer(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
		sessionManager: newSessionManager(redis),
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		movieRepo:      movieRepo,
		theaterRepo:    theaterRepo,
		seatRepo:       seatRepo,
	}

	return app.run()
}

func newSessionManager(pool *redis.Pool) *scs.SessionManager {
	sessionManager := scs.New()

	sessionManager.Store = redisstore.New(pool)
	sessionManager.IdleTimeout = 20 * time.Minute
	sessionManager.Cookie.Name = "session_id"

	return sessionManager
}

func newRedisPool(cfg config) (*redis.Pool, error) {
	pool := &redis.Pool{
		MaxIdle:     cfg.redis.maxIdleConns,
		MaxActive:   cfg.redis.maxOpenConns,
		IdleTimeout: cfg.redis.maxIdleTime,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", cfg.redis.url)
		},
	}

	conn := pool.Get()
	defer conn.Close()

	_, err := conn.Do("PING")
	if err != nil {
		fmt.Printf("Redis connection error: %s", err.Error())
		pool.Close()
		return nil, err
	}

	return pool, nil
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
		Addr:         fmt.Sprintf("127.0.0.1:%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
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

	return api.HandlerFromMux(app, r)
}
