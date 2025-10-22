package handlers

import (
	"net/http"
	"time"
)

var startedAt = time.Now()

// HealthHandler — простой liveness/probe.
// Если захочешь readiness (проверка БД/Кафки) — добавим позже.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"status":     "ok",
		"service":    "order-service",
		"started_at": startedAt.Format(time.RFC3339),
		"uptime_sec": int(time.Since(startedAt).Seconds()),
	}
	writeJSON(w, http.StatusOK, resp)
}
