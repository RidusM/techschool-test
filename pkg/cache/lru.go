package cache

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"wbtest/pkg/logger"
	"wbtest/pkg/metric"
)

const (
	_removePreallocSize = 10
)

type LRUCache[K comparable, V any] struct {
	cache   map[K]*list.Element
	lruList *list.List
	mutex   sync.Mutex
	log     logger.Logger
	metrics metric.Cache

	capacity        int
	cleanupInterval time.Duration
	cleanupStop     chan struct{}
	onEvicted       func(key K, value V)
}

type entry[K comparable, V any] struct {
	key     K
	value   V
	expires time.Time
}

func NewLRUCache[K comparable, V any](
	capacity int,
	log logger.Logger,
	metrics metric.Cache,
) (*LRUCache[K, V], error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("cache.NewLRUCache: capacity must be positive, got %d", capacity)
	}

	return &LRUCache[K, V]{
		capacity: capacity,
		cache:    make(map[K]*list.Element),
		lruList:  list.New(),
		log:      log,
		metrics:  metrics,
	}, nil
}

func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	var zero V

	c.mutex.Lock()
	defer c.mutex.Unlock()

	elem, ok := c.cache[key]
	if !ok {
		c.metrics.Miss("order")
		return zero, false
	}

	entry, ok := elem.Value.(*entry[K, V])
	if !ok {
		c.log.Errorw("cache contains value of unexpected type",
			"type", fmt.Sprintf("%T", elem.Value),
		)
		c.removeElement(elem)
		c.metrics.Miss("order")
		return zero, false
	}

	if !entry.expires.IsZero() && time.Now().After(entry.expires) {
		c.removeElement(elem)
		c.metrics.Miss("order")
		return zero, false
	}

	c.lruList.MoveToFront(elem)
	c.metrics.Hit("order")

	return entry.value, true
}

func (c *LRUCache[K, V]) Put(key K, value V, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var expires time.Time

	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}

	if elem, ok := c.cache[key]; ok {
		if entry, exist := elem.Value.(*entry[K, V]); exist {
			c.lruList.MoveToFront(elem)
			entry.value = value
			entry.expires = expires
			return
		}
		c.lruList.Remove(elem)
		delete(c.cache, key)
	}

	if c.lruList.Len() >= c.capacity {
		c.removeOldest()
	}

	e := &entry[K, V]{
		key:     key,
		value:   value,
		expires: expires,
	}
	elem := c.lruList.PushFront(e)
	c.cache[key] = elem
}

func (c *LRUCache[K, V]) Has(key K) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	elem, ok := c.cache[key]
	if !ok {
		return false
	}

	entry, ok := elem.Value.(*entry[K, V])
	if !ok {
		return false
	}

	return entry.expires.IsZero() || time.Now().Before(entry.expires)
}

func (c *LRUCache[K, V]) Len() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.lruList.Len()
}

func (c *LRUCache[K, V]) Capacity() int {
	return c.capacity
}

func (c *LRUCache[K, V]) Purge() {
	var evicted []struct {
		key   K
		value V
	}

	c.mutex.Lock()
	for key, elem := range c.cache {
		if entry, ok := elem.Value.(*entry[K, V]); ok {
			evicted = append(evicted, struct {
				key   K
				value V
			}{key, entry.value})
		}
	}
	c.lruList.Init()
	clear(c.cache)
	c.mutex.Unlock()

	for _, item := range evicted {
		if c.onEvicted != nil {
			c.onEvicted(item.key, item.value)
		}
	}
}

func (c *LRUCache[K, V]) StartCleanup(interval time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.cleanupStop != nil {
		close(c.cleanupStop)
	}

	c.cleanupInterval = interval
	c.cleanupStop = make(chan struct{})
	go c.runCleanup()
}

func (c *LRUCache[K, V]) StopCleanup() {
	c.mutex.Lock()
	if c.cleanupStop != nil {
		close(c.cleanupStop)
		c.cleanupStop = nil
	}
	c.mutex.Unlock()
}

func (c *LRUCache[K, V]) runCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.cleanupStop:
			return
		}
	}
}

func (c *LRUCache[K, V]) cleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	removed := 0
	toRemove := make([]*list.Element, 0, _removePreallocSize)

	for _, elem := range c.cache {
		entry, ok := elem.Value.(*entry[K, V])
		if !ok {
			continue
		}

		if entry.expires.IsZero() {
			continue
		}

		if now.After(entry.expires) {
			toRemove = append(toRemove, elem)
		}
	}

	for _, elem := range toRemove {
		c.removeElement(elem)
		removed++
	}

	if removed > 0 {
		c.log.Infow("cache cleanup completed",
			"removed", removed,
			"remaining", c.lruList.Len(),
		)
	}
}

func (c *LRUCache[K, V]) removeOldest() {
	if elem := c.lruList.Back(); elem != nil {
		c.removeElement(elem)
	}
}

func (c *LRUCache[K, V]) removeElement(elem *list.Element) {
	c.lruList.Remove(elem)
	entry, ok := elem.Value.(*entry[K, V])
	if !ok {
		c.log.Errorw("cache contains value of unexpected type",
			"type", fmt.Sprintf("%T", elem.Value),
		)
		return
	}
	delete(c.cache, entry.key)
	if c.onEvicted != nil {
		c.onEvicted(entry.key, entry.value)
	}
	c.metrics.Eviction("order", "lru")
}

func (c *LRUCache[K, V]) SetOnEvicted(onEvicted func(key K, value V)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.onEvicted = onEvicted
}
