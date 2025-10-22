package services

import (
	"context"
	"errors"
	"github.com/reybrally/order-service/internal/adapters/cache"
	"github.com/reybrally/order-service/internal/app/orders"
	domain "github.com/reybrally/order-service/internal/domain/order"
	"github.com/reybrally/order-service/internal/validation"
)

type OrderService struct {
	repo         orders.OrderRepo
	cacheService cache.Cache
}

func NewOrderService(repo orders.OrderRepo, cache cache.Cache) *OrderService {
	return &OrderService{repo: repo, cacheService: cache}
}

func (service *OrderService) CreateOrUpdateOrder(ctx context.Context, o domain.Order) (domain.Order, error) {

	if err := validation.IsValidOrder(o); err != nil {
		return domain.Order{}, err
	}

	ord, err := service.repo.CreateOrUpdateOrder(ctx, o)
	if err != nil {
		return domain.Order{}, err
	}
	_ = service.cacheService.Set(o.OrderUID, o)
	return ord, nil

}
func (service *OrderService) GetOrder(ctx context.Context, id string) (domain.Order, error) {

	if ord, err := service.cacheService.Get(id); err == nil {
		return ord, nil
	}
	ord, err := service.repo.GetOrder(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	service.cacheService.Set(id, ord)
	return ord, nil

}

func (service *OrderService) DeleteOrder(ctx context.Context, id string) error {

	if id == "" {
		return errors.New("id is required")
	}

	if err := service.repo.DeleteOrder(ctx, id); err != nil {
		return err
	}
	service.cacheService.Delete(id)
	return nil
}

// отдает список заказов по фильтрам и пагинации
func (service *OrderService) SearchOrder(ctx context.Context, filters orders.SearchFilters, req orders.PageRequest) ([]domain.Order, error) {
	orders.NormalizeSearchFilters(&filters)
	orders.NormalizeRequest(&req)
	return service.repo.SearchOrders(ctx, filters, req)
}

/* helpers */
