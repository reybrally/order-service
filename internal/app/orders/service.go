package orders

import (
	"context"
	"errors"
	domain "github.com/reybrally/order-service/internal/domain/order"
	"github.com/reybrally/order-service/internal/validation"
)

type OrderService struct {
	repo OrderRepo
}

func NewOrderService(repo OrderRepo) *OrderService {
	return &OrderService{repo: repo}
}

func (service *OrderService) CreateOrUpdateOrder(ctx context.Context, o domain.Order) (domain.Order, error) {

	if err := validation.IsValidOrder(o); err != nil {
		return domain.Order{}, err
	}
	return service.repo.CreateOrUpdateOrder(ctx, o)
}
func (service *OrderService) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	return service.repo.GetOrder(ctx, id)
}

func (service *OrderService) DeleteOrder(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("id is required")
	}
	return service.repo.DeleteOrder(ctx, id)
}

// отдает список заказов по фильтрам и пагинации
func (s *OrderService) SearchOrder(ctx context.Context, filters SearchFilters, req PageRequest) ([]domain.Order, error) {
	NormalizeSearchFilters(&filters)
	NormalizeRequest(&req)
	return s.repo.SearchOrders(ctx, filters, req)
}

/* helpers */
