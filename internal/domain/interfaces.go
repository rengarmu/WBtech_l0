package domain

import (
	"context"
	"time"
)

// CacheStats — статистика использования кеша
type CacheStats struct {
	Hits   int64
	Misses int64
	Size   int
}

// OrderCache определяет методы для работы с кешем заказов
type OrderCache interface {
	Get(orderUID string) (Order, bool)
	Set(order Order)
	SetWithTTL(order Order, ttl time.Duration)
	Delete(orderUID string)
	Clear()
	GetAll() map[string]Order
	GetStats() CacheStats
}

// OrderRepository определяет методы для работы с БД
type OrderRepository interface {
	SaveOrder(ctx context.Context, order Order) error
	GetOrder(ctx context.Context, orderUID string) (Order, error)
	LoadAllOrders(ctx context.Context) ([]Order, error)
	ClearAll(ctx context.Context) error
}

// OrderUsecase объединяет бизнес-логику получения и сохранения заказов
type OrderUsecase interface {
	GetOrder(ctx context.Context, orderUID string) (Order, error)
	SaveOrder(ctx context.Context, order Order) error
}
