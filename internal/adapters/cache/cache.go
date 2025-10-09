package cache

import (
	"github.com/reybrally/order-service/internal/app/orders"
	"github.com/reybrally/order-service/internal/domain/order"
)

type lruNode struct {
	key   string
	value order.Order
	next  *lruNode
	prev  *lruNode
}

type LRUCache struct {
	head     *lruNode // наименее актуальная нода
	tail     *lruNode // наиболиее актуальная нода
	cache    map[string]*lruNode
	capacity int
}

func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		head:     nil,
		tail:     nil,
		cache:    make(map[string]*lruNode),
		capacity: capacity,
	}
}

// Set устанавливает новое значение существующей ноде либо же добавляет новую к существующим
func (c *LRUCache) Set(key string, value order.Order) error {
	if nd, ok := c.cache[key]; ok {
		c.cache[key].value = value
		c.changeTail(nd)
		return nil
	}
	nd := newLruNode(key, value)
	if c.capacity == len(c.cache) {
		delete(c.cache, c.head.key)
		c.deleteNode(c.head)

	}
	c.addCache(nd)
	c.cache[nd.key] = nd
	return nil

}

func (c *LRUCache) Get(key string) (order.Order, error) {
	nd, ok := c.cache[key]
	if !ok {
		return order.Order{}, orders.ErrNotFound
	}
	c.changeTail(nd)
	return nd.value, nil

}

func (c *LRUCache) Clear() {
	c.cache = make(map[string]*lruNode)
	c.head = nil
	c.tail = nil
}

// helpers

// newLruNode создает новую ноду кеша
func newLruNode(key string, value order.Order) *lruNode {
	return &lruNode{
		key:   key,
		value: value,
	}
}

// addCache добавляет ноду в список нод лру кеша
func (c *LRUCache) addCache(nd *lruNode) {
	if c.head == nil {
		c.head = nd
		c.tail = nd
		return
	}
	c.tail.next = nd
	nd.prev = c.tail
	c.tail = nd
}

func (c *LRUCache) deleteNode(nd *lruNode) {
	if nd == nil {
		return
	}
	if nd == c.head {
		c.head = c.head.next
		c.head.prev = nil
		return
	}
	if nd == c.tail {
		nd.prev.next = nil
		c.tail = nd.prev
		nd.prev = nil
		return
	}
	nd.prev.next = nd.next
	nd.next.prev = nd.prev
	nd = nil
}

func (c *LRUCache) changeTail(nd *lruNode) {
	if c.head == nd && c.tail == nd {
		return
	}
	c.deleteNode(nd)
	c.addCache(nd)
}
