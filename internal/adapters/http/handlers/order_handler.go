package handlers

import (
	"context"
	"encoding/json"
	"github.com/reybrally/order-service/internal/app/orders"
	"github.com/reybrally/order-service/internal/domain/order"
	"net/http"
)

type OrderHandlers struct {
	svc serviceInterface
}

type serviceInterface interface {
	CreateOrUpdateOrder(ctx context.Context, o order.Order) (order.Order, error)
	GetOrder(ctx context.Context, id string) (order.Order, error)
	DeleteOrder(ctx context.Context, id string) error
	SearchOrder(ctx context.Context, filters orders.SearchFilters, req orders.PageRequest) ([]order.Order, error)
}

func NewOrderHandlers(svc serviceInterface) *OrderHandlers {
	return &OrderHandlers{svc: svc}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, errorResponse{Error: msg})
}

// ToResponseList — helper для массива
func ToResponseList(src []order.Order) []OrderResponse {
	out := make([]OrderResponse, 0, len(src))
	for _, o := range src {
		out = append(out, ToResponse(o))
	}
	return out
}

func strptr(s string) *string { return &s }
