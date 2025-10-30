package handlers

import (
	"encoding/json"
	"github.com/reybrally/order-service/internal/adapters/http/handlers/validation"
	"github.com/reybrally/order-service/internal/logging"
	"github.com/sirupsen/logrus"
	"net/http"
)

func (h *OrderHandlers) CreateOrUpdateOrder(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	var req OrderUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logging.LogError("Error decoding request body", err, logrus.Fields{"method": "CreateOrUpdateOrder"})
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	logging.LogInfo("Request body decoded", logrus.Fields{"method": "CreateOrUpdateOrder"})
	order, err := req.ToModel()
	if err != nil {
		logging.LogError("Error converting to model", err, logrus.Fields{"method": "CreateOrUpdateOrder"})
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := validation.IsValidOrder(order); err != nil {
		logging.LogError("Invalid order", err, logrus.Fields{"method": "CreateOrUpdateOrder"})
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ctx := r.Context()

	ord, err := h.svc.CreateOrUpdateOrder(ctx, order)
	if err != nil {
		logging.LogError("Error creating or updating order", err, logrus.Fields{"method": "CreateOrUpdateOrder"})
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	status := http.StatusOK
	if req.OrderUID == nil || *req.OrderUID == "" {
		status = http.StatusCreated
		w.Header().Set("Location", "/orders/"+ord.OrderUID)
	}

	logging.LogInfo("Order created or updated", logrus.Fields{"method": "CreateOrUpdateOrder", "order_uid": ord.OrderUID})
	writeJSON(w, status, ToResponse(ord))

}
