package cache

import (
	"sync"

	"github.com/reybrally/order-service/internal/app/orders"
	"github.com/reybrally/order-service/internal/domain/order"
)

type lruNode struct {
	key   string
	value order.Order
	prev  *lruNode
	next  *lruNode
}

// CacheService — потокобезопасный LRU.
type CacheService struct {
	mu       sync.Mutex
	cache    map[string]*lruNode
	head     *lruNode // наименее актуальная (LRA)
	tail     *lruNode // наиболее актуальная (MRU)
	capacity int
}

func NewCacheService(capacity int) *CacheService {
	if capacity <= 0 {
		capacity = 1
	}
	return &CacheService{
		cache:    make(map[string]*lruNode, capacity),
		capacity: capacity,
	}
}

// Set: вставить/обновить и пометить как MRU.
func (c *CacheService) Set(key string, value order.Order) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Обновление существующей ноды
	if nd, ok := c.cache[key]; ok {
		nd.value = value
		c.moveToTail(nd)
		return nil
	}

	// Эвикт при заполнении
	if len(c.cache) >= c.capacity {
		c.evictHead() // корректно удалит и из списка, и из map
	}

	// Вставка новой ноды в хвост (MRU)
	nd := &lruNode{key: key, value: value}
	c.appendToTail(nd)
	c.cache[key] = nd
	return nil
}

// Get: получить и пометить как MRU. На промах — orders.ErrNotFound.
func (c *CacheService) Get(key string) (order.Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	nd, ok := c.cache[key]
	if !ok {
		return order.Order{}, orders.ErrNotFound
	}
	c.moveToTail(nd)
	return nd.value, nil
}

// Delete: удалить ключ. Если нет — orders.ErrNotFound.
func (c *CacheService) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	nd, ok := c.cache[key]
	if !ok {
		return orders.ErrNotFound
	}
	c.unlink(nd)
	delete(c.cache, key)
	return nil
}

func (c *CacheService) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*lruNode, c.capacity)
	c.head = nil
	c.tail = nil
}

// Доп. удобства (по желанию)
func (c *CacheService) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.cache)
}
func (c *CacheService) Cap() int { return c.capacity }

// ===== внутренняя работа со списком =====

func (c *CacheService) appendToTail(nd *lruNode) {
	if c.tail == nil { // пустой список
		c.head = nd
		c.tail = nd
		return
	}
	nd.prev = c.tail
	c.tail.next = nd
	c.tail = nd
}

func (c *CacheService) moveToTail(nd *lruNode) {
	if nd == c.tail { // уже MRU
		return
	}
	// сначала отцепить из текущей позиции
	c.unlink(nd)
	// а затем приложить к хвосту
	c.appendToTail(nd)
}

func (c *CacheService) evictHead() {
	if c.head == nil {
		return
	}
	evicted := c.head
	c.unlink(evicted)
	delete(c.cache, evicted.key)
}

func (c *CacheService) unlink(nd *lruNode) {
	if nd == nil {
		return
	}

	// связываем соседей между собой
	if nd.prev != nil {
		nd.prev.next = nd.next
	} else {
		// nd был головой
		c.head = nd.next
	}
	if nd.next != nil {
		nd.next.prev = nd.prev
	} else {
		// nd был хвостом
		c.tail = nd.prev
	}

	// зануляем ссылки ноды (на всякий случай)
	nd.prev = nil
	nd.next = nil
}
