package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"WBtech_l0/internal/config"
	"WBtech_l0/internal/domain"
	"WBtech_l0/internal/repository/cache"

	_ "github.com/lib/pq"
)

// InitDB — подключение к PostgreSQL
func InitDB(cfg config.Config) *sql.DB {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Database)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	return db
}

// LoadCacheFromDB — восстанавливает кеш из БД при старте
func LoadCacheFromDB(db *sql.DB, cache *cache.OrderCache) error {
	rows, err := db.Query("SELECT order_uid FROM orders")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return err
		}
		order, err := GetOrderFromDB(db, orderUID)
		if err == nil {
			cache.Set(order)
		}
	}
	return nil
}

// SaveOrderTx — сохраняет заказ в БД в транзакции (атомарно)
func SaveOrderTx(db *sql.DB, order domain.Order) error {
	// Валидация заказа перед сохранением
	if err := order.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Если что-то пойдет не так — откат
	defer tx.Rollback()

	// Вставляем основной заказ
	_, err = tx.Exec(`
        INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature,
                            customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        ON CONFLICT (order_uid) DO NOTHING`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale,
		order.InternalSignature, order.CustomerID, order.DeliveryService,
		order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		return err
	}

	// Вставляем доставку
	_, err = tx.Exec(`
        INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone,
		order.Delivery.Zip, order.Delivery.City, order.Delivery.Address,
		order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		return err
	}

	// Вставляем оплату
	_, err = tx.Exec(`
        INSERT INTO payments (order_uid, transaction, request_id, currency, provider,
                              amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID,
		order.Payment.Currency, order.Payment.Provider, order.Payment.Amount,
		order.Payment.PaymentDT, order.Payment.Bank, order.Payment.DeliveryCost,
		order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		return err
	}

	// Вставляем товары
	for _, item := range order.Items {
		_, err = tx.Exec(`
            INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name,
                               sale, size, total_price, nm_id, brand, status)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price,
			item.Rid, item.Name, item.Sale, item.Size, item.TotalPrice,
			item.NmID, item.Brand, item.Status)
		if err != nil {
			return err
		}
	}

	// Фиксируем транзакцию
	return tx.Commit()
}

// GetOrderFromDB — достает заказ по order_uid
func GetOrderFromDB(db *sql.DB, orderUID string) (domain.Order, error) {
	var order domain.Order

	// Проверяем валидность orderUID
	if orderUID == "" || len(orderUID) > 100 {
		return order, fmt.Errorf("invalid order_uid format")
	}

	// Начинаем транзакцию с уровнем изоляции Repeatable Read
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		return order, err
	}
	defer tx.Rollback() // Safe to call if tx is committed

	// Получаем данные заказа в транзакции
	err = tx.QueryRow(`
        SELECT order_uid, track_number, entry, locale, internal_signature,
               customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders WHERE order_uid=$1`, orderUID).Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale,
		&order.InternalSignature, &order.CustomerID, &order.DeliveryService,
		&order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)
	if err != nil {
		return order, err
	}

	// Доставка
	err = tx.QueryRow(`
        SELECT name, phone, zip, city, address, region, email
        FROM deliveries WHERE order_uid=$1`, orderUID).Scan(
		&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
		&order.Delivery.City, &order.Delivery.Address,
		&order.Delivery.Region, &order.Delivery.Email)
	if err != nil {
		return order, err
	}

	// Оплата
	err = tx.QueryRow(`
        SELECT transaction, request_id, currency, provider,
               amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
        FROM payments WHERE order_uid=$1`, orderUID).Scan(
		&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
		&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDT,
		&order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal,
		&order.Payment.CustomFee)
	if err != nil {
		return order, err
	}

	// Товары
	rows, err := tx.Query(`
        SELECT chrt_id, track_number, price, rid, name, sale, size,
               total_price, nm_id, brand, status
        FROM items WHERE order_uid=$1`, orderUID)
	if err != nil {
		return order, err
	}
	defer rows.Close()

	var items []domain.Item
	for rows.Next() {
		var it domain.Item
		err := rows.Scan(&it.ChrtID, &it.TrackNumber, &it.Price, &it.Rid, &it.Name,
			&it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status)
		if err != nil {
			return order, err
		}
		items = append(items, it)
	}
	order.Items = items

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return order, err
	}

	return order, nil
}
