// Package cache реализует in-memory кеширование заказов
package cache

import (
	"sync"
	"time"

	"WBtech_l0/internal/domain" // импортируем domain для использования domain.CacheStats
)

// OrderCache — in-memory кеш заказов
type OrderCache struct {
	mu         sync.RWMutex
	items      map[string]Item
	defaultTTL time.Duration
	maxSize    int
	stats      domain.CacheStats // используем domain.CacheStats
}

// Item — элемент кеша с TTL
type Item struct {
	Order     domain.Order
	ExpiresAt time.Time
}

// NewOrderCache создаёт новый кеш
func NewOrderCache(defaultTTL time.Duration, maxSize int) *OrderCache {
	c := &OrderCache{
		items:      make(map[string]Item),
		defaultTTL: defaultTTL,
		maxSize:    maxSize,
	}
	go c.cleanupExpired()
	return c
}

// SetWithTTL добавляет заказ с указанным временем жизни
func (c *OrderCache) SetWithTTL(order domain.Order, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[order.OrderUID] = Item{
		Order:     order,
		ExpiresAt: time.Now().Add(ttl),
	}
	c.stats.Size = len(c.items)
}

// Set добавляет заказ с TTL по умолчанию
func (c *OrderCache) Set(order domain.Order) {
	c.SetWithTTL(order, c.defaultTTL)
}

// Get возвращает заказ из кеша
func (c *OrderCache) Get(orderUID string) (domain.Order, bool) {
	c.mu.RLock()
	item, exists := c.items[orderUID]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return domain.Order{}, false
	}

	if time.Now().After(item.ExpiresAt) {
		c.Delete(orderUID)
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return domain.Order{}, false
	}

	c.mu.Lock()
	c.stats.Hits++
	c.mu.Unlock()

	return item.Order, true
}

// GetAll возвращает все неистекшие заказы
func (c *OrderCache) GetAll() map[string]domain.Order {
	c.mu.RLock()
	defer c.mu.RUnlock()

	orders := make(map[string]domain.Order)
	now := time.Now()
	for uid, item := range c.items {
		if now.Before(item.ExpiresAt) {
			orders[uid] = item.Order
		}
	}
	return orders
}

// Delete удаляет заказ
func (c *OrderCache) Delete(orderUID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, orderUID)
	c.stats.Size = len(c.items)
}

// Clear очищает кеш
func (c *OrderCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]Item)
	c.stats.Size = 0
	c.stats.Hits = 0
	c.stats.Misses = 0
}

// GetStats возвращает статистику (теперь возвращает domain.CacheStats)
func (c *OrderCache) GetStats() domain.CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// cleanupExpired периодически удаляет просроченные записи
func (c *OrderCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for uid, item := range c.items {
			if now.After(item.ExpiresAt) {
				delete(c.items, uid)
			}
		}
		c.stats.Size = len(c.items)
		c.mu.Unlock()
	}
}

// evictOldest удаляет самую старую запись при переполнении
func (c *OrderCache) evictOldest() {
	var oldestUID string
	var oldestTime time.Time
	for uid, item := range c.items {
		if oldestTime.IsZero() || item.ExpiresAt.Before(oldestTime) {
			oldestTime = item.ExpiresAt
			oldestUID = uid
		}
	}
	if oldestUID != "" {
		delete(c.items, oldestUID)
	}
}
