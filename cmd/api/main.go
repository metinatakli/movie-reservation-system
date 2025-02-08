package main

import (
	"os"

	"github.com/metinatakli/movie-reservation-system/internal/app"
)

func main() {
	err := app.Run()
	if err != nil {
		os.Exit(1)
	}
}
