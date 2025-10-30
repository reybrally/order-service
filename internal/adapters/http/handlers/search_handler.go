package handlers

import (
	"github.com/reybrally/order-service/internal/adapters/http/handlers/normalization"
	"github.com/reybrally/order-service/internal/logging"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"

	"github.com/reybrally/order-service/internal/app/orders"
)

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
			logging.LogError("Invalid 'created_from' query parameter", err, logrus.Fields{"method": "SearchOrders"})
			writeError(w, http.StatusBadRequest, "invalid created_from (RFC3339 expected)")
			return
		}
	}
	if s := q.Get("created_to"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			f.CreatedTo = &t
		} else {
			logging.LogError("Invalid 'created_to' query parameter", err, logrus.Fields{"method": "SearchOrders"})
			writeError(w, http.StatusBadRequest, "invalid created_to (RFC3339 expected)")
			return
		}
	}

	f.OrderUID = strptr(q.Get("order_uid"))
	f.TrackNumber = strptr(q.Get("track_number"))
	f.CustomerID = strptr(q.Get("customer_id"))
	f.Provider = strptr(q.Get("provider"))
	f.Currency = strptr(q.Get("currency"))
	f.Query = strptr(q.Get("q"))

	normalization.NormalizeSearchFilters(&f)

	normalization.NormalizeRequest(&p)

	logging.LogDebug("Search filters", logrus.Fields{"method": "SearchOrders", "filters": f})
	logging.LogDebug("Page request", logrus.Fields{"method": "SearchOrders", "page_request": p})

	ctx := r.Context()
	list, err := h.svc.SearchOrder(ctx, f, p)
	if err != nil {
		logging.LogError("Error searching orders", err, logrus.Fields{"method": "SearchOrders"})
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	logging.LogInfo("Orders found", logrus.Fields{"method": "SearchOrders", "count": len(list)})

	writeJSON(w, http.StatusOK, ToResponseList(list))
}
