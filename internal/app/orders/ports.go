package orders

import (
	"context"
	domain "github.com/reybrally/order-service/internal/domain/order"
	"time"
)

type OrderCreatorUpdater interface {
	CreateOrUpdateOrder(ctx context.Context, o domain.Order) (domain.Order, error)
}

type OrderGetter interface {
	GetOrder(ctx context.Context, id string) (domain.Order, error)
}

type OrderDeleter interface {
	DeleteOrder(ctx context.Context, id string) error
}

type OrderSearcher interface {
	SearchOrders(ctx context.Context, filters SearchFilters, request PageRequest) ([]domain.Order, error)
}

type SearchFilters struct {
	CreatedFrom *time.Time
	CreatedTo   *time.Time

	OrderUID    *string
	TrackNumber *string
	CustomerID  *string
	Provider    *string
	Currency    *string

	Query *string
}

type PageRequest struct {
	Limit   int
	Offset  int
	SortBy  string
	SortDir string
}

type OrderRepo interface {
	OrderCreatorUpdater
	OrderGetter
	OrderDeleter
	OrderSearcher
}
