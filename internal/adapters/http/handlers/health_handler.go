package handlers

import (
	"github.com/reybrally/order-service/internal/logging"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

var startedAt = time.Now()

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	logging.LogInfo("Health check requested", logrus.Fields{"method": "HealthHandler"})
	resp := map[string]any{
		"status":     "ok",
		"service":    "order-service",
		"started_at": startedAt.Format(time.RFC3339),
		"uptime_sec": int(time.Since(startedAt).Seconds()),
	}
	writeJSON(w, http.StatusOK, resp)
}
