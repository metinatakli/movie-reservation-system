package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/metinatakli/movie-reservation-system/internal/config"
	"github.com/metinatakli/movie-reservation-system/internal/handler"
	"github.com/metinatakli/movie-reservation-system/internal/vcs"
)

var (
	version = vcs.Version()
)

type application struct {
	config   config.Config
	logger   *slog.Logger
	handlers *handlers
}

type handlers struct {
	handler.HealthcheckHandler
}

func main() {
	var cfg config.Config

	flag.IntVar(&cfg.Port, "port", 3000, "server port")
	flag.StringVar(&cfg.Env, "env", "dev", "Environment (dev|staging|prod)")

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	app := application{
		config: cfg,
		logger: logger,
		handlers: &handlers{
			HealthcheckHandler: *handler.NewHealthcheckHandler(cfg),
		},
	}

	err := app.run()
	if err != nil {
		os.Exit(1)
	}
}
