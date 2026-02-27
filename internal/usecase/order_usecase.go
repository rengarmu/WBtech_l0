// Package usecase содержит бизнес-логику приложения
package usecase

import (
	"context"
	"fmt"

	"WBtech_l0/internal/domain"
)

type orderUsecase struct {
	repo  domain.OrderRepository
	cache domain.OrderCache
}

// NewOrderUsecase создаёт новый экземпляр usecase
func NewOrderUsecase(repo domain.OrderRepository, cache domain.OrderCache) domain.OrderUsecase {
	return &orderUsecase{
		repo:  repo,
		cache: cache,
	}
}

// GetOrder сначала ищет в кеше, затем в БД и сохраняет в кеш
func (u *orderUsecase) GetOrder(ctx context.Context, orderUID string) (domain.Order, error) {
	// Пробуем из кеша
	if order, found := u.cache.Get(orderUID); found {
		return order, nil
	}

	// Из БД
	order, err := u.repo.GetOrder(ctx, orderUID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("repo.GetOrder: %w", err)
	}

	// Сохраняем в кеш
	u.cache.Set(order)
	return order, nil
}

// SaveOrder сохраняет в БД и обновляет кеш
func (u *orderUsecase) SaveOrder(ctx context.Context, order domain.Order) error {
	if err := u.repo.SaveOrder(ctx, order); err != nil {
		return err
	}
	u.cache.Set(order)
	return nil
}
