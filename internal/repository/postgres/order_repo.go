// Package postgres реализует репозиторий для работы с PostgreSQL
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"WBtech_l0/internal/config"
	"WBtech_l0/internal/domain"
	"WBtech_l0/internal/repository/cache"

	"github.com/lib/pq"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// DBPinger - интерфейс для проверки соединения с БД
type DBPinger interface {
	PingContext(ctx context.Context) error
}

// Repository — реализация доменного репозитория для PostgreSQL
type Repository struct {
	db *sql.DB
}

// InitDB — подключение к PostgreSQL
func InitDB(cfg config.Config) *Repository {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Database)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	return &Repository{db: db}
}

// Close — закрытие соединения с БД
func (r *Repository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// PingContext реализует интерфейс DBPinger для health check
func (r *Repository) PingContext(ctx context.Context) error {
	if err := r.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err) // обёрнуто
	}
	return nil
}

// LoadCacheFromDB — восстанавливает кеш из БД при старте
func LoadCacheFromDB(ctx context.Context, r *Repository, cache *cache.OrderCache) error {
	rows, err := r.db.QueryContext(ctx, "SELECT order_uid FROM orders")
	if err != nil {
		return fmt.Errorf("query order uids: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			return fmt.Errorf("scan order_uid: %w", err)
		}
		order, err := r.GetOrder(ctx, orderUID)
		if err == nil {
			cache.Set(order)
		} else {
			log.Printf("Failed to load order %s from DB: %v", orderUID, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows error: %w", err)
	}
	return nil
}

// SaveOrder — сохраняет заказ в БД в транзакции (атомарно)
func (r *Repository) SaveOrder(ctx context.Context, order domain.Order) error {
	// Валидация заказа перед сохранением
	if err := order.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Если что-то пойдет не так — откат
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback transaction: %v", err)
		}
	}()

	// Вставляем основной заказ
	_, err = tx.ExecContext(ctx, `
        INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature,
                            customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        ON CONFLICT (order_uid) DO NOTHING`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale,
		order.InternalSignature, order.CustomerID, order.DeliveryService,
		order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	// Вставляем доставку
	_, err = tx.ExecContext(ctx, `
        INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone,
		order.Delivery.Zip, order.Delivery.City, order.Delivery.Address,
		order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		return fmt.Errorf("insert delivery: %w", err)
	}

	// Вставляем оплату
	_, err = tx.ExecContext(ctx, `
        INSERT INTO payments (order_uid, transaction, request_id, currency, provider,
                              amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID,
		order.Payment.Currency, order.Payment.Provider, order.Payment.Amount,
		order.Payment.PaymentDT, order.Payment.Bank, order.Payment.DeliveryCost,
		order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	// Вставляем товары
	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, `
            INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name,
                               sale, size, total_price, nm_id, brand, status)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price,
			item.Rid, item.Name, item.Sale, item.Size, item.TotalPrice,
			item.NmID, item.Brand, item.Status)
		if err != nil {
			return fmt.Errorf("insert item: %w", err)
		}
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// GetOrder — достает заказ по order_uid
func (r *Repository) GetOrder(ctx context.Context, orderUID string) (domain.Order, error) {
	var order domain.Order

	// Проверяем валидность orderUID
	if orderUID == "" || len(orderUID) > 100 {
		return order, fmt.Errorf("invalid order_uid format")
	}

	// Начинаем транзакцию с уровнем изоляции Repeatable Read
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		return order, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback transaction: %v", err)
		}
	}()

	// Получаем данные заказа в транзакции
	err = tx.QueryRowContext(ctx, `
        SELECT order_uid, track_number, entry, locale, internal_signature,
               customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders WHERE order_uid=$1`, orderUID).Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale,
		&order.InternalSignature, &order.CustomerID, &order.DeliveryService,
		&order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)
	if err != nil {
		return order, fmt.Errorf("query delivery: %w", err)
	}

	// Доставка
	err = tx.QueryRowContext(ctx, `
        SELECT name, phone, zip, city, address, region, email
        FROM deliveries WHERE order_uid=$1`, orderUID).Scan(
		&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
		&order.Delivery.City, &order.Delivery.Address,
		&order.Delivery.Region, &order.Delivery.Email)
	if err != nil {
		return order, fmt.Errorf("query delivery: %w", err)
	}

	// Оплата
	err = tx.QueryRowContext(ctx, `
        SELECT transaction, request_id, currency, provider,
               amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
        FROM payments WHERE order_uid=$1`, orderUID).Scan(
		&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
		&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDT,
		&order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal,
		&order.Payment.CustomFee)
	if err != nil {
		return order, fmt.Errorf("query payment: %w", err)
	}

	// Товары
	rows, err := tx.QueryContext(ctx, `
        SELECT chrt_id, track_number, price, rid, name, sale, size,
               total_price, nm_id, brand, status
        FROM items WHERE order_uid=$1`, orderUID)
	if err != nil {
		return order, fmt.Errorf("query items: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	var items []domain.Item
	for rows.Next() {
		var it domain.Item
		err := rows.Scan(&it.ChrtID, &it.TrackNumber, &it.Price, &it.Rid, &it.Name,
			&it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status)
		if err != nil {
			return order, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return order, fmt.Errorf("items iteration: %w", err) // FIXED: wrapped error
	}
	order.Items = items

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return order, fmt.Errorf("commit transaction: %w", err)
	}

	return order, nil
}

// LoadAllOrders загружает все заказы из БД со связанными данными
func (r *Repository) LoadAllOrders(ctx context.Context) ([]domain.Order, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback transaction: %v", err)
		}
	}()

	// 1. Загружаем основные данные всех заказов
	rows, err := tx.QueryContext(ctx, `
        SELECT order_uid, track_number, entry, locale, internal_signature,
               customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders
        ORDER BY date_created DESC
    `)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close orders rows: %v", err)
		}
	}()

	var orders []domain.Order
	orderIdxMap := make(map[string]int) // order_uid -> индекс в слайсе orders

	for rows.Next() {
		var o domain.Order
		err := rows.Scan(
			&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale,
			&o.InternalSignature, &o.CustomerID, &o.DeliveryService,
			&o.Shardkey, &o.SmID, &o.DateCreated, &o.OofShard,
		)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orderIdxMap[o.OrderUID] = len(orders)
		orders = append(orders, o)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if len(orders) == 0 {
		return orders, nil
	}

	// Получаем список всех order_uid для массовой загрузки связанных таблиц
	uids := make([]string, 0, len(orders))
	for _, o := range orders {
		uids = append(uids, o.OrderUID)
	}

	// 2. Загружаем доставки
	deliveryRows, err := tx.QueryContext(ctx, `
        SELECT order_uid, name, phone, zip, city, address, region, email
        FROM deliveries
        WHERE order_uid = ANY($1)
    `, pq.Array(uids))
	if err != nil {
		return nil, fmt.Errorf("query deliveries: %w", err)
	}
	defer func() {
		if err := deliveryRows.Close(); err != nil {
			log.Printf("failed to close deliveries rows: %v", err)
		}
	}()

	for deliveryRows.Next() {
		var orderUID string
		var d domain.Delivery
		err := deliveryRows.Scan(
			&orderUID, &d.Name, &d.Phone, &d.Zip, &d.City, &d.Address, &d.Region, &d.Email,
		)
		if err != nil {
			return nil, fmt.Errorf("scan delivery: %w", err)
		}
		if idx, ok := orderIdxMap[orderUID]; ok {
			orders[idx].Delivery = d
		}
	}
	if err = deliveryRows.Err(); err != nil {
		return nil, fmt.Errorf("deliveries iteration: %w", err)
	}

	// 3. Загружаем платежи
	paymentRows, err := tx.QueryContext(ctx, `
        SELECT order_uid, transaction, request_id, currency, provider,
               amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
        FROM payments
        WHERE order_uid = ANY($1)
    `, pq.Array(uids))
	if err != nil {
		return nil, fmt.Errorf("query payments: %w", err)
	}
	defer func() {
		if err := paymentRows.Close(); err != nil {
			log.Printf("failed to close payments rows: %v", err)
		}
	}()

	for paymentRows.Next() {
		var orderUID string
		var p domain.Payment
		err := paymentRows.Scan(
			&orderUID, &p.Transaction, &p.RequestID, &p.Currency, &p.Provider,
			&p.Amount, &p.PaymentDT, &p.Bank, &p.DeliveryCost, &p.GoodsTotal, &p.CustomFee,
		)
		if err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		if idx, ok := orderIdxMap[orderUID]; ok {
			orders[idx].Payment = p
		}
	}
	if err = paymentRows.Err(); err != nil {
		return nil, fmt.Errorf("payments iteration: %w", err)
	}

	// 4. Загружаем товары
	itemRows, err := tx.QueryContext(ctx, `
        SELECT order_uid, chrt_id, track_number, price, rid, name,
               sale, size, total_price, nm_id, brand, status
        FROM items
        WHERE order_uid = ANY($1)
        ORDER BY order_uid, chrt_id
    `, pq.Array(uids))
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer func() {
		if err := itemRows.Close(); err != nil {
			log.Printf("failed to close items rows: %v", err)
		}
	}()

	for itemRows.Next() {
		var orderUID string
		var it domain.Item
		err := itemRows.Scan(
			&orderUID, &it.ChrtID, &it.TrackNumber, &it.Price, &it.Rid, &it.Name,
			&it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		if idx, ok := orderIdxMap[orderUID]; ok {
			orders[idx].Items = append(orders[idx].Items, it)
		}
	}
	if err = itemRows.Err(); err != nil {
		return nil, fmt.Errorf("items iteration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return orders, nil
}

// ClearAll удаляет все записи из таблиц и сбрасывает последовательности
func (r *Repository) ClearAll(ctx context.Context) error {
	tables := []string{"items", "payments", "deliveries", "orders"}
	for _, table := range tables {
		if _, err := r.db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}
	}
	// Сброс последовательностей (опционально)
	sequences := []string{"items_id_seq", "payments_id_seq", "deliveries_id_seq"}
	for _, seq := range sequences {
		if _, err := r.db.ExecContext(ctx, "ALTER SEQUENCE "+seq+" RESTART WITH 1"); err != nil {
			log.Printf("failed to restart sequence %s: %v", seq, err)
		}
	}
	return nil
}
