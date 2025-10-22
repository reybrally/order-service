package services

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/reybrally/order-service/internal/adapters/cache"
	"github.com/reybrally/order-service/internal/app/orders"
	domain "github.com/reybrally/order-service/internal/domain/order"
)

type OrderService struct {
	repo         orders.OrderRepo
	cacheService cache.Cache
}

func NewOrderService(repo orders.OrderRepo, cache cache.Cache) *OrderService {
	return &OrderService{repo: repo, cacheService: cache}
}

func (serv *OrderService) CreateOrUpdateOrder(ctx context.Context, or domain.Order) (domain.Order, error) {

	if or.OrderUID == "" {
		or.OrderUID = uuid.New().String()
	}

	ord, err := serv.repo.CreateOrUpdateOrder(ctx, or)
	if err != nil {
		return domain.Order{}, err
	}
	_ = serv.cacheService.Set(ord.OrderUID, ord)
	return ord, nil

}
func (serv *OrderService) GetOrder(ctx context.Context, id string) (domain.Order, error) {

	if ord, err := serv.cacheService.Get(id); err == nil {
		return ord, nil
	}
	ord, err := serv.repo.GetOrder(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	serv.cacheService.Set(id, ord)
	return ord, nil

}

func (serv *OrderService) DeleteOrder(ctx context.Context, id string) error {

	if id == "" {
		return errors.New("id is required")
	}

	if err := serv.repo.DeleteOrder(ctx, id); err != nil {
		return err
	}
	_ = serv.cacheService.Delete(id)
	return nil
}

// отдает список заказов по фильтрам и пагинации
func (serv *OrderService) SearchOrder(ctx context.Context, filters orders.SearchFilters, req orders.PageRequest) ([]domain.Order, error) {
	orders.NormalizeSearchFilters(&filters)
	orders.NormalizeRequest(&req)
	return serv.repo.SearchOrders(ctx, filters, req)
}

/* helpers */
