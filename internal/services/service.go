package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/reybrally/order-service/internal/adapters/cache"
	kaf "github.com/reybrally/order-service/internal/adapters/kafka"
	"github.com/reybrally/order-service/internal/app/orders"
	domain "github.com/reybrally/order-service/internal/domain/order"
	"github.com/reybrally/order-service/internal/logging"
)

type OrderService struct {
	repo         orders.OrderRepo
	cacheService cache.Cache

	// Kafka
	producer    kaf.Producer
	eventsTopic string
}

func NewOrderService(repo orders.OrderRepo, cache cache.Cache, producer kaf.Producer, eventsTopic string) *OrderService {
	return &OrderService{
		repo:         repo,
		cacheService: cache,
		producer:     producer,
		eventsTopic:  eventsTopic,
	}
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

	// Твоя текущая семантика кэша — сохраняем
	_ = serv.cacheService.Set(ord.OrderUID, ord)
	logging.LogInfo("Order created or updated successfully", logrus.Fields{"order_uid": ord.OrderUID})

	// Публикуем событие в Kafka (cache projector/другие подписчики)
	env := kaf.Envelope[kaf.OrderUpserted]{
		EventType:  "order.upserted",
		Version:    1,
		OccurredAt: time.Now().UTC(),
		EntityID:   ord.OrderUID,
		Payload:    kaf.OrderUpserted{OrderUID: ord.OrderUID},
		Meta:       kaf.Meta{Producer: "order-service", Source: "http"},
	}
	if serv.producer != nil && serv.eventsTopic != "" {
		if err := kaf.PublishEnvelope(ctx, serv.producer, serv.eventsTopic, []byte(ord.OrderUID), env, nil); err != nil {
			logging.LogError("Failed to publish order.upserted", err, logrus.Fields{"order_uid": ord.OrderUID})
		}
	}

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
		err := errors.New("id is required")
		logging.LogError("ID is required for deletion", err, logrus.Fields{"method": "DeleteOrder"})
		return err
	}

	if err := serv.repo.DeleteOrder(ctx, id); err != nil {
		logging.LogError("Error deleting order from repository", err, logrus.Fields{"order_uid": id})
		return err
	}

	_ = serv.cacheService.Delete(id)
	logging.LogInfo("Order deleted successfully", logrus.Fields{"order_uid": id})

	env := kaf.Envelope[kaf.OrderDeleted]{
		EventType:  "order.deleted",
		Version:    1,
		OccurredAt: time.Now().UTC(),
		EntityID:   id,
		Payload:    kaf.OrderDeleted{OrderUID: id},
		Meta:       kaf.Meta{Producer: "order-service", Source: "http"},
	}
	if serv.producer != nil && serv.eventsTopic != "" {
		if err := kaf.PublishEnvelope(ctx, serv.producer, serv.eventsTopic, []byte(id), env, nil); err != nil {
			logging.LogError("Failed to publish order.deleted", err, logrus.Fields{"order_uid": id})
		}
	}

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
