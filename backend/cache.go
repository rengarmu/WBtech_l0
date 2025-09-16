package backend

import "sync"

// OrderCache — кеш заказов в памяти
type OrderCache struct {
	mu     sync.RWMutex
	orders map[string]Order
}

// Создаём новый кеш
func NewOrderCache() *OrderCache {
	return &OrderCache{
		orders: make(map[string]Order),
	}
}

// Добавляем заказ в кеш
func (c *OrderCache) Set(order Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.orders[order.OrderUID] = order
}

// Получаем заказ из кеша
func (c *OrderCache) Get(orderUID string) (Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	order, exists := c.orders[orderUID]
	return order, exists
}
