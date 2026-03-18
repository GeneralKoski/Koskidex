package engine

import (
	"container/list"
	"sync"
)

type cacheEntry struct {
	key   string
	value interface{}
}

type LRUCache struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return el.Value.(*cacheEntry).value, true
	}
	return nil, false
}

func (c *LRUCache) Put(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		el.Value.(*cacheEntry).value = value
		return
	}

	if c.order.Len() >= c.capacity {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.items, oldest.Value.(*cacheEntry).key)
		}
	}

	entry := &cacheEntry{key: key, value: value}
	el := c.order.PushFront(entry)
	c.items[key] = el
}

func (c *LRUCache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var toDelete []string
	for key := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			toDelete = append(toDelete, key)
		}
	}
	for _, key := range toDelete {
		if el, ok := c.items[key]; ok {
			c.order.Remove(el)
			delete(c.items, key)
		}
	}
}

func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.order.Init()
}
