package handlers

import (
	"encoding/json"
	"github.com/reybrally/order-service/internal/validation"
	"net/http"
)

func (h *OrderHandlers) CreateOrUpdateOrder(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	var req OrderUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	order, err := req.ToModel()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := validation.IsValidOrder(order); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ctx := r.Context()

	ord, err := h.svc.CreateOrUpdateOrder(ctx, order)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	status := http.StatusOK
	if req.OrderUID == nil || *req.OrderUID == "" {
		status = http.StatusCreated
		w.Header().Set("Location", "/orders/"+ord.OrderUID)
	}

	writeJSON(w, status, ToResponse(ord))

}
