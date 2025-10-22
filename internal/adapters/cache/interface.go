package cache

import domain "github.com/reybrally/order-service/internal/domain/order"

type Cache interface {
	Set(key string, o domain.Order) error
	Get(key string) (domain.Order, error)
	Delete(key string) error
}
