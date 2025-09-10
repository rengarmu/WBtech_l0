package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
	"wb_tech/back/models"

	config "WB_tech/back/config"
	model "WB_tech/back/models"
)

// Database - обертка для работы с базой данных PostgreSQL
type Database struct {
	db *sql.DB
}

// NewDatabaseConnect создает новое подключение к PostgreSQL
func NewDatabaseConnect(cfg *config.PsqlConfig) (*Database, error) {
	cfgStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)
	db, error := sql.Open("postgres", cfgStr)
	if error != nil {
		return nil, fmt.Errorf("failed to open database: %s", error)
	}
	if error = db.Ping(); error != nil {
		return nil, fmt.Errorf("failed to ping database: %s", error)
	}
	db.SetMaxOpenConns(25)                 // макс. одновременно открытых соединений
	db.SetMaxIdleConns(25)                 // макс. соединений в пуле бездействия
	db.SetConnMaxLifetime(5 * time.Minute) // макс. время жизни соединения
	return &Database{db: db}, nil
}

// CreateOrder сохраняет заказ в базу данных
func (p *Database) CreateOrder(ctx context.Context, order *models.Order) error {
	text, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if errRB := text.Rollback(); err != nil {
				log.Printf("failed to rollback transaction: %v", errRB)
			}
		}
	}()

	insertOrder, err := text.PrepareContext(ctx,
		`INSERT INTO orders(order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created,oof_shard)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`)
	if err != nil {
		if err != nil {
			return fmt.Errorf("failed to prepare order: %w", err)
		}
	}
	defer insertOrder.Close()

	insertDelivery, err := text.PrepareContext(ctx,
		`INSERT INTO delivery(order_uid, name, phone, zip, city, address, region, email
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`)
	if err != nil {
		return fmt.Errorf("failed to prepare delivery: %w", err)
	}
	defer insertDelivery.Close()

	insertPayment, err := text.PrepareContext(ctx, `
	INSERT INTO payment (order_uid, transaction_number, request_id VARCHAR(255), currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUE ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`)
	if err != nil {
		return fmt.Errorf("failed to prepare payment: %w", err)
	}
	defer insertPayment.Close()

	insertItem, err := text.PrepareContext(ctx, `
	INSERT INTO items (order_uid, chrt_id, track_number, price, rid, item_name, sale, size, total_price, nm_id, brand, status)
		VALUE ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`)
	if err != nil {
		return fmt.Errorf("failed to prepare items: %w", err)
	}
	defer insertItem.Close()
	_, err = insertOrder.ExecContext(ctx,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature, order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	_, err = insertDelivery.ExecContext(ctx, order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip, order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		return fmt.Errorf("failed to insert delivery: %w", err)
	}

	_, err = insertPayment.ExecContext(ctx, order.OrderUID, order.Payment.TransactionNumder, order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDT, order.Payment.Bank, order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		return fmt.Errorf("failed to insert payment: %w", err)
	}
	for _, item := range order.Items {
		_, err = insertItem.ExecContext(ctx, order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.RID, item.ItemName, item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			return fmt.Errorf("failed to insert item: %w", err)
		}
	}
	err = text.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// GetOrder достаёт данные о заказе из базы данных по order_uid
func (p *Database) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	querySql := `
	SELECT order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created,oof_shard
	FROM orders
	WHERE order_uid = $1`
	order := &model.Order{}
	err := p.db.QueryRowContext(ctx, querySql, orderUID).Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature, &order.CustomerID, &order.DeliveryService,
		&order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, fmt.Errorf("failed to get order: %w", err)
		}
	}
	querySql = `SELECT name, phone, zip, city, address, region, email
	FROM delivery
	WHERE order_uid = $1`
	err = p.db.QueryRowContext(ctx, querySql, orderUID).Scan(&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
		&order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, fmt.Errorf("failed to get delivery: %w", err)
		}
	}
	querySql = `SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
	FROM payment
	WHERE order_uid = $1`
	err = p.db.QueryRowContext(ctx, querySql, orderUID).Scan(&order.Payment.TransactionNumder, &order.Payment.RequestID, &order.Payment.Currency,
		&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDT, &order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal,
		&order.Payment.CustomFee)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, fmt.Errorf("failed to get payment: %w", err)
		}
	}
	querySql = `SELECT chrt_id, track_number, price, rid, item_name, sale, size, total_price, nm_id, brand, status
	FROM items
	WHERE order_uid = $1`
	rows, err := p.db.QueryContext(ctx, querySql, orderUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		item := model.Item{}
		err = rows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.RID, &item.ItemName, &item.Sale, &item.Size, &item.TotalPrice,
			&item.NmID, &item.Brand, &item.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		order.Items = append(order.Items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan items: %w", err)
	}
	return order, nil
}

// GetAllOrders извлекает все заказы из базы данных
func (p *Database) GetAllOrders(ctx context.Context) ([]*model.Order, error) {
	querySql := `
        SELECT order_uid, track_number, entry, locale, internal_signature, customer_id,
        delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders
    `
	rows, err := p.db.QueryContext(ctx, querySql)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders from database: %w", err)
	}
	defer rows.Close()

	orders := []*model.Order{}
	for rows.Next() {
		order := &model.Order{}
		err := rows.Scan(
			&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
			&order.CustomerID, &order.DeliveryService, &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		// err = p.populateRelatedData(ctx, order)
		order, err = p.GetOrder(ctx, order.OrderUID)

		if err != nil {
			return nil, fmt.Errorf("failed to populate related data for order %s: %w", order.OrderUID, err)
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}

	return orders, nil
}

func (p *Database) Close() error {
	if err := p.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}
