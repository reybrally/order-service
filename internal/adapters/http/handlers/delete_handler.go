package handlers

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/reybrally/order-service/internal/app/orders"
	"github.com/reybrally/order-service/internal/logging"
	"github.com/sirupsen/logrus"
	"net/http"
)

func (h *OrderHandlers) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		logging.LogError("ID is required in DeleteHandler", nil, logrus.Fields{"method": "DeleteHandler", "id": id})
		writeError(w, http.StatusBadRequest, "id can't be empty")
		return
	}

	logging.LogInfo("Attempting to delete order", logrus.Fields{"method": "DeleteHandler", "id": id})
	ctx := r.Context()
	err := h.svc.DeleteOrder(ctx, id)
	if err != nil {
		if errors.Is(err, orders.ErrNotFound) {
			logging.LogError("Order not found", err, logrus.Fields{"method": "DeleteHandler", "id": id})
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		logging.LogError("Error deleting order", err, logrus.Fields{"method": "DeleteHandler", "id": id})
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	logging.LogInfo("Order deleted successfully", logrus.Fields{"method": "DeleteHandler", "id": id})
	writeJSON(w, http.StatusOK, nil)
}
