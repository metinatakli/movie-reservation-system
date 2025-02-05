package handler

import (
	"encoding/json"
	"net/http"

	"github.com/metinatakli/movie-reservation-system/api"
	"github.com/metinatakli/movie-reservation-system/internal/config"
	"github.com/metinatakli/movie-reservation-system/internal/vcs"
)

type HealthcheckHandler struct {
	cfg config.Config
}

func NewHealthcheckHandler(cfg config.Config) *HealthcheckHandler {
	return &HealthcheckHandler{
		cfg: cfg,
	}
}

func (h *HealthcheckHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	status := "UP"
	systemInfo := api.SystemInfo{
		Version:     vcs.Version(),
		Environment: h.cfg.Env,
	}

	resp := api.HealthcheckResponse{
		Status:     status,
		SystemInfo: systemInfo,
	}

	json.NewEncoder(w).Encode(resp)
}
