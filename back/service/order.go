package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	cache "wb_tech/back/cash"
	model "wb_tech/back/models"
)

// OrderService содержит бизнес-логику обработки заказов
type OrderService struct {
	cache *cache.Cache
	db    *sql.DB
}

// NewOrderService создает новый сервис заказов
func NewOrderService(cache *cache.Cache, db *sql.DB) *OrderService {
	return &OrderService{cache: cache, db: db}
}

// ProcessOrder обрабатывает новый заказ
func (s *OrderService) ProcessOrder(order model.Order) error {
	ctx := context.Background()

	if err := s.db.QueryRowContext(ctx, &order); err != nil {
		return fmt.Errorf("failed to process order: %w", err)
	}

	s.cache.Set(order)
	log.Println("Order saved to database and cache:", order)

	return nil
}

// RestoreCacheFromDB восстанавливает кэш из базы данных при запуске
func (s *OrderService) RestoreCacheFromDB() error {
	ctx := context.Background()

	orders, err := s.db.GetAllOrders(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all orders from database: %w", err)
	}

	s.cache.LoadFromSlice(orders)
	log.Println("Cache restored from database with", len(orders))

	return nil
}
