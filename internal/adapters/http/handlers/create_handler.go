package handlers

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/reybrally/order-service/internal/app/orders"
	"net/http"
)

func (h *OrderHandlers) GetHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	ctx := r.Context()
	order, err := h.svc.GetOrder(ctx, id)
	if err != nil {
		if errors.Is(err, orders.ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ToResponse(order))

}
