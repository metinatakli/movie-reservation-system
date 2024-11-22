package main

import (
	"flag"
	"log/slog"
	"os"
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

	flag.Parse()

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
