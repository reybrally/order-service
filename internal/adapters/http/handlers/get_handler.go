package handlers

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/reybrally/order-service/internal/app/orders"
	"github.com/reybrally/order-service/internal/logging"
	"github.com/sirupsen/logrus"
	"net/http"
)

func (h *OrderHandlers) GetHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		logging.LogError("ID is required in GetHandler", nil, logrus.Fields{"method": "GetHandler", "id": id})
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	logging.LogInfo("Fetching order", logrus.Fields{"method": "GetHandler", "id": id})

	ctx := r.Context()
	order, err := h.svc.GetOrder(ctx, id)
	if err != nil {
		if errors.Is(err, orders.ErrNotFound) {
			logging.LogError("Order not found", err, logrus.Fields{"method": "GetHandler", "id": id})
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		logging.LogError("Internal server error while fetching order", err, logrus.Fields{"method": "GetHandler", "id": id})
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	logging.LogInfo("Order found", logrus.Fields{"method": "GetHandler", "id": id})
	writeJSON(w, http.StatusOK, ToResponse(order))

}
