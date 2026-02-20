package cache

import (
	"sync"
	"time"

	"WBtech_l0/internal/domain"
)

// OrderCache — кеш заказов в памяти
type OrderCache struct {
	mu         sync.RWMutex
	items      map[string]CacheItem
	defaultTTL time.Duration
	maxSize    int
	stats      CacheStats
}

// CacheItem представляет элемент кэша с временем жизни
type CacheItem struct {
	Order     domain.Order
	ExpiresAt time.Time
}

// CacheStats содержит статистику использования кэша
type CacheStats struct {
	Hits   int64
	Misses int64
	Size   int
}

// Создаём новый кеш
func NewOrderCache() *OrderCache {
	cache := &OrderCache{
		items:      make(map[string]CacheItem),
		defaultTTL: 20 * time.Minute, // Время жизни по умолчанию
		maxSize:    1000,             // Максимальный размер кэша
	}
	// Фоновая очистка устаревших записей
	go cache.cleanupExpired()
	return cache

}

// Устанавливаем максимальный размер кэша
func (c *OrderCache) SetMaxSize(size int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxSize = size
}

// Устанавливаем TTL по умолчанию
func (c *OrderCache) SetDefaultTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaultTTL = ttl
}

// Добавляем заказ в кеш с TTL по умолчанию
func (c *OrderCache) Set(order domain.Order) {
	c.SetWithTTL(order, c.defaultTTL)
}

// Добавляем заказ в кеш с указанным временем жизни
func (c *OrderCache) SetWithTTL(order domain.Order, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Проверяем размер кэша и удаляем старые записи при необходимости
	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[order.OrderUID] = CacheItem{
		Order:     order,
		ExpiresAt: time.Now().Add(ttl),
	}
	c.stats.Size = len(c.items)
}

// Получаем заказ из кеша
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

	// Проверяем, не истекло ли время жизни
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

// Возвращаем все неистекшие заказы из кеша
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

// Удаляем заказ из кеша
func (c *OrderCache) Delete(orderUID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, orderUID)
	c.stats.Size = len(c.items)
}

// Очищаем кеш полностью
func (c *OrderCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]CacheItem)
	c.stats.Size = 0
	c.stats.Hits = 0
	c.stats.Misses = 0
}

// Возвращаем статистику использования кэша
func (c *OrderCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// Периодическое удаление устаревших записей
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

// Удаляем самую старую запись при переполнении кэша
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
