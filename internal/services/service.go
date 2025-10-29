package services

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/reybrally/order-service/internal/adapters/cache"
	"github.com/reybrally/order-service/internal/app/orders"
	domain "github.com/reybrally/order-service/internal/domain/order"
	"github.com/reybrally/order-service/internal/logging"
	"github.com/sirupsen/logrus"
)

type OrderService struct {
	repo         orders.OrderRepo
	cacheService cache.Cache
}

func NewOrderService(repo orders.OrderRepo, cache cache.Cache) *OrderService {
	return &OrderService{repo: repo, cacheService: cache}
}

func (serv *OrderService) CreateOrUpdateOrder(ctx context.Context, or domain.Order) (domain.Order, error) {

	logging.LogInfo("Attempting to create or update order", logrus.Fields{"order_uid": or.OrderUID})
	if or.OrderUID == "" {
		or.OrderUID = uuid.New().String()
	}

	ord, err := serv.repo.CreateOrUpdateOrder(ctx, or)
	if err != nil {
		logging.LogError("Error during CreateOrUpdateOrder", err, logrus.Fields{"order_uid": or.OrderUID})
		return domain.Order{}, err
	}
	_ = serv.cacheService.Set(ord.OrderUID, ord)
	logging.LogInfo("Order created or updated successfully", logrus.Fields{"order_uid": ord.OrderUID})
	return ord, nil

}
func (serv *OrderService) GetOrder(ctx context.Context, id string) (domain.Order, error) {

	logging.LogInfo("Fetching order", logrus.Fields{"order_uid": id})

	if ord, err := serv.cacheService.Get(id); err == nil {
		logging.LogInfo("Order found in cache", logrus.Fields{"order_uid": id})
		return ord, nil
	}
	ord, err := serv.repo.GetOrder(ctx, id)
	if err != nil {
		logging.LogError("Error fetching order from repository", err, logrus.Fields{"order_uid": id})

		return domain.Order{}, err
	}
	_ = serv.cacheService.Set(id, ord)
	logging.LogInfo("Order fetched and cached", logrus.Fields{"order_uid": id})
	return ord, nil

}

func (serv *OrderService) DeleteOrder(ctx context.Context, id string) error {
	logging.LogInfo("Attempting to delete order", logrus.Fields{"order_uid": id})

	if id == "" {
		logging.LogError("ID is required for deletion", nil, logrus.Fields{"method": "DeleteOrder"})
		return errors.New("id is required")
	}

	if err := serv.repo.DeleteOrder(ctx, id); err != nil {
		logging.LogError("Error deleting order from repository", err, logrus.Fields{"order_uid": id})
		return err
	}
	_ = serv.cacheService.Delete(id)
	logging.LogInfo("Order deleted successfully", logrus.Fields{"order_uid": id})
	return nil
}

// SearchOrder отдает список заказов по фильтрам и пагинации
func (serv *OrderService) SearchOrder(ctx context.Context, filters orders.SearchFilters, req orders.PageRequest) ([]domain.Order, error) {
	logging.LogInfo("Searching for orders", logrus.Fields{"filters": filters, "page_request": req})

	list, err := serv.repo.SearchOrders(ctx, filters, req)
	if err != nil {
		logging.LogError("Error searching orders in repository", err, logrus.Fields{"filters": filters, "page_request": req})
		return nil, err
	}

	logging.LogInfo("Orders found", logrus.Fields{"count": len(list)})
	return list, nil
}
