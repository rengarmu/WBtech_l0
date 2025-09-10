package cache

import (
	"sync"
	"wb_tech/back/models"
)

// Cache представляет in-memory кэш заказов
type Cache struct {
	mu     sync.RWMutex
	orders map[string]models.Order
}

// New создает новый экземпляр кэша
// Инициализирует пустую map для хранения заказов
func New() *Cache {
	return &Cache{
		orders: make(map[string]models.Order),
	}
}

// Set добавляет или обновляет заказ в кэше
func (c *Cache) Set(order models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.orders[order.OrderUID] = order
}

// Get возвращает заказ из кэша по order_uid
func (c *Cache) Get(orderUID string) (models.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	order, exists := c.orders[orderUID]
	return order, exists
}

// GetAll возвращает все заказы из кэша
func (c *Cache) GetAll() []models.Order {
	c.mu.RLock()
	defer c.mu.RUnlock()

	orders := make([]models.Order, 0, len(c.orders))
	for _, order := range c.orders {
		orders = append(orders, order)
	}

	return orders
}

// LoadFromSlice полностью заменяет содержимое кэша
func (c *Cache) LoadFromSlice(orders []models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.orders = make(map[string]models.Order)
	for _, order := range orders {
		c.orders[order.OrderUID] = order
	}
}

// Size возвращает количество заказов в кэше
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.orders)
}
