package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/reybrally/order-service/internal/app/orders"
)

// SearchOrders — GET /orders/search

func (h *OrderHandlers) SearchOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var (
		f orders.SearchFilters
		p orders.PageRequest
	)

	if s := q.Get("created_from"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			f.CreatedFrom = &t
		} else {
			writeError(w, http.StatusBadRequest, "invalid created_from (RFC3339 expected)")
			return
		}
	}
	if s := q.Get("created_to"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			f.CreatedTo = &t
		} else {
			writeError(w, http.StatusBadRequest, "invalid created_to (RFC3339 expected)")
			return
		}
	}

	if s := q.Get("order_uid"); s != "" {
		f.OrderUID = strptr(s)
	}
	if s := q.Get("track_number"); s != "" {
		f.TrackNumber = strptr(s)
	}
	if s := q.Get("customer_id"); s != "" {
		f.CustomerID = strptr(s)
	}
	if s := q.Get("provider"); s != "" {
		f.Provider = strptr(s)
	}
	if s := q.Get("currency"); s != "" {
		f.Currency = strptr(s)
	}

	if s := q.Get("q"); s != "" {
		f.Query = strptr(s)
	}

	if s := q.Get("sort_by"); s != "" {
		p.SortBy = s
	}
	if s := q.Get("sort_dir"); s != "" {
		p.SortDir = s
	}
	if s := q.Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			p.Limit = v
		} else {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}
	if s := q.Get("offset"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			p.Offset = v
		} else {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
	}

	ctx := r.Context()
	list, err := h.svc.SearchOrder(ctx, f, p)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ToResponseList(list))
}
