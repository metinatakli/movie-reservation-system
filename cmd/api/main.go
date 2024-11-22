package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/metinatakli/movie-reservation-system/internal/vcs"
)

var (
	version = vcs.Version()
)

type config struct {
	port int
	env  string
}

type application struct {
	config config
	logger *slog.Logger
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 3000, "server port")
	flag.StringVar(&cfg.env, "env", "dev", "Environment (dev|staging|prod)")

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
	}

	err := app.run()
	if err != nil {
		os.Exit(1)
	}
}
