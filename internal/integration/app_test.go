package integration_test

import (
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metinatakli/movie-reservation-system/internal/app"
	"github.com/metinatakli/movie-reservation-system/internal/mailer"
	"github.com/metinatakli/movie-reservation-system/internal/payment"
	"github.com/metinatakli/movie-reservation-system/internal/repository"
	appvalidator "github.com/metinatakli/movie-reservation-system/internal/validator"
)

type TestApp struct {
	App    *app.Application
	DB     *pgxpool.Pool
	Mailer *mailer.MockMailer
}

func newTestApp(cfg app.Config) (*TestApp, error) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	validator := appvalidator.NewValidator()
	mailer := mailer.NewMockMailer()

	db, err := app.NewDatabasePool(cfg)
	if err != nil {
		return nil, err
	}

	redisClient, err := app.NewRedisClient(cfg)
	if err != nil {
		db.Close()
		return nil, err
	}

	sessionManager := app.NewSessionManager(redisClient)

	userRepo := repository.NewPostgresUserRepository(db)
	tokenRepo := repository.NewPostgresTokenRepository(db)
	movieRepo := repository.NewPostgresMovieRepository(db)
	theaterRepo := repository.NewPostgresTheaterRepository(db)
	seatRepo := repository.NewPostgresSeatRepository(db)
	paymentRepo := repository.NewPostgresPaymentRepository(db)
	reservationRepo := repository.NewPostgresReservationRepository(db)

	paymentProvider := payment.NewMockPaymentProvider()

	application := app.NewApp(
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
		paymentProvider,
	)

	return &TestApp{
		App:    application,
		DB:     db,
		Mailer: mailer,
	}, nil
}
