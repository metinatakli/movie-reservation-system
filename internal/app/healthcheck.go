package app

import (
	"net/http"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/vcs"
)

func (app *Application) GetHealth(w http.ResponseWriter, r *http.Request) {
	status := "UP"
	systemInfo := api.SystemInfo{
		Version:     vcs.Version(),
		Environment: app.config.Env,
	}

	resp := api.HealthcheckResponse{
		Status:     status,
		SystemInfo: systemInfo,
	}

	app.writeJSON(w, http.StatusOK, resp, nil)
}
