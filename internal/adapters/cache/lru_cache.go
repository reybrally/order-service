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

type CacheService struct {
	mu       sync.Mutex
	cache    map[string]*lruNode
	head     *lruNode
	tail     *lruNode
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

func (c *CacheService) Set(key string, value order.Order) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if nd, ok := c.cache[key]; ok {
		nd.value = value
		c.moveToTail(nd)
		return nil
	}

	if len(c.cache) >= c.capacity {
		c.evictHead()
	}

	nd := &lruNode{key: key, value: value}
	c.appendToTail(nd)
	c.cache[key] = nd
	return nil
}

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

func (c *CacheService) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.cache)
}
func (c *CacheService) Cap() int { return c.capacity }

func (c *CacheService) appendToTail(nd *lruNode) {
	if c.tail == nil {
		c.head = nd
		c.tail = nd
		return
	}
	nd.prev = c.tail
	c.tail.next = nd
	c.tail = nd
}

func (c *CacheService) moveToTail(nd *lruNode) {
	if nd == c.tail {
		return
	}
	c.unlink(nd)
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

	if nd.prev != nil {
		nd.prev.next = nd.next
	} else {
		c.head = nd.next
	}
	if nd.next != nil {
		nd.next.prev = nd.prev
	} else {
		c.tail = nd.prev
	}

	nd.prev = nil
	nd.next = nil
}
