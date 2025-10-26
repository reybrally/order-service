package handlers

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/reybrally/order-service/internal/app/orders"
	"net/http"
)

func (h *OrderHandlers) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id can't be empty")
		return
	}
	ctx := r.Context()
	err := h.svc.DeleteOrder(ctx, id)
	if err != nil {
		if errors.Is(err, orders.ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nil)
}
