package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"WBtech_l0/internal/domain"

	_ "github.com/lib/pq"
)

// ensureTestDBExists создаёт тестовую базу данных, если она не существует.
func ensureTestDBExists() error {
	connStr := "host=localhost port=5432 user=tmp password=test90123 dbname=orders_db_test sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to test database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close test DB connection: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("test database 'orders_db_test' does not exist or is not accessible.\n" +
			"Please create it manually:\n" +
			"  sudo -u postgres psql -c \"CREATE DATABASE orders_db_test OWNER tmp;\"\n" +
			"Then apply migrations:\n" +
			"  migrate -database 'postgres://tmp:test90123@localhost:5432/orders_db_test?sslmode=disable' -path migrations up")
	}
	return nil
}

func TestMain(m *testing.M) {
	if err := ensureTestDBExists(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// connectTestDB подключается к тестовой базе данных.
func connectTestDB(t *testing.T) *sql.DB {
	host := "localhost"
	port := "5432"
	user := "tmp"
	password := "test90123"
	dbname := "orders_db_test"
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}
	if err = db.PingContext(context.Background()); err != nil {
		t.Fatalf("failed to ping test DB: %v", err)
	}
	return db
}

// truncateTables очищает таблицы между тестами.
func truncateTables(t *testing.T, db *sql.DB) {
	tables := []string{"items", "payments", "deliveries", "orders"}
	for _, table := range tables {
		_, err := db.ExecContext(context.Background(), "DELETE FROM "+table)
		if err != nil {
			t.Fatalf("failed to truncate %s: %v", table, err)
		}
	}
}

func TestPostgresRepository_SaveAndGetOrder(t *testing.T) {
	db := connectTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close test database: %v", err)
		}
	}()
	truncateTables(t, db)

	repo := &Repository{db: db}

	ctx := context.Background()
	order := domain.Order{
		OrderUID:    "test-save-1",
		TrackNumber: "TRACK123",
		Entry:       "WBIL",
		Delivery: domain.Delivery{
			Name:    "John Doe",
			Phone:   "+123456789",
			Zip:     "12345",
			City:    "Test City",
			Address: "Test Address",
			Region:  "Test Region",
			Email:   "test@example.com",
		},
		Payment: domain.Payment{
			Transaction:  "test-trx",
			Currency:     "USD",
			Provider:     "test",
			Amount:       1000,
			PaymentDT:    time.Now().Unix(),
			Bank:         "Test Bank",
			DeliveryCost: 100,
			GoodsTotal:   900,
		},
		Items: []domain.Item{
			{
				ChrtID:      1,
				TrackNumber: "TRACK123",
				Price:       500,
				Rid:         "rid1",
				Name:        "Item 1",
				Sale:        0,
				Size:        "M",
				TotalPrice:  500,
				NmID:        100,
				Brand:       "Brand",
				Status:      200,
			},
		},
		Locale:          "en",
		CustomerID:      "cust1",
		DeliveryService: "test-delivery",
		Shardkey:        "1",
		SmID:            1,
		DateCreated:     time.Now().Format(time.RFC3339),
		OofShard:        "1",
	}

	err := repo.SaveOrder(ctx, order)
	if err != nil {
		t.Fatalf("SaveOrder failed: %v", err)
	}

	saved, err := repo.GetOrder(ctx, "test-save-1")
	if err != nil {
		t.Fatalf("GetOrder failed: %v", err)
	}

	if saved.OrderUID != order.OrderUID {
		t.Errorf("OrderUID mismatch: expected %s, got %s", order.OrderUID, saved.OrderUID)
	}
	if saved.Delivery.Name != order.Delivery.Name {
		t.Errorf("Delivery.Name mismatch")
	}
	if saved.Payment.Amount != order.Payment.Amount {
		t.Errorf("Payment.Amount mismatch")
	}
	if len(saved.Items) != len(order.Items) {
		t.Errorf("Items count mismatch")
	}
}

func TestPostgresRepository_GetOrder_NotFound(t *testing.T) {
	db := connectTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close test database: %v", err)
		}
	}()
	truncateTables(t, db)

	repo := &Repository{db: db}
	_, err := repo.GetOrder(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent order, got nil")
	}
}

func TestPostgresRepository_LoadAllOrders(t *testing.T) {
	db := connectTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close test database: %v", err)
		}
	}()
	truncateTables(t, db)

	repo := &Repository{db: db}
	ctx := context.Background()

	now := time.Now().Format(time.RFC3339)
	order1 := domain.Order{
		OrderUID:    "load1",
		TrackNumber: "T1",
		Entry:       "WBIL",
		Delivery: domain.Delivery{
			Name:    "D1",
			Phone:   "1",
			Zip:     "1",
			City:    "C",
			Address: "A",
			Region:  "R",
			Email:   "e@e.com",
		},
		Payment: domain.Payment{
			Transaction: "trx1",
			Currency:    "USD",
			Amount:      100,
			PaymentDT:   time.Now().Unix(),
		},
		Items: []domain.Item{
			{
				ChrtID:      1,
				TrackNumber: "T1",
				Price:       10,
				Name:        "I1",
				TotalPrice:  10,
				NmID:        1,
			},
		},
		DateCreated: now,
	}
	order2 := domain.Order{
		OrderUID:    "load2",
		TrackNumber: "T2",
		Entry:       "WBIL",
		Delivery: domain.Delivery{
			Name:    "D2",
			Phone:   "2",
			Zip:     "2",
			City:    "C2",
			Address: "A2",
			Region:  "R2",
			Email:   "e2@e.com",
		},
		Payment: domain.Payment{
			Transaction: "trx2",
			Currency:    "USD",
			Amount:      200,
			PaymentDT:   time.Now().Unix(),
		},
		Items: []domain.Item{
			{
				ChrtID:      2,
				TrackNumber: "T2",
				Price:       20,
				Name:        "I2",
				TotalPrice:  20,
				NmID:        2,
			},
		},
		DateCreated: now,
	}

	if err := repo.SaveOrder(ctx, order1); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveOrder(ctx, order2); err != nil {
		t.Fatal(err)
	}

	// Отладочная проверка наличия связанных данных в БД
	var deliveryCount, itemCount int

	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM deliveries WHERE order_uid=$1", "load1").Scan(&deliveryCount); err != nil {
		t.Fatal(err)
	}
	t.Logf("deliveries for load1: %d", deliveryCount)

	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM items WHERE order_uid=$1", "load1").Scan(&itemCount); err != nil {
		t.Fatal(err)
	}
	t.Logf("items for load1: %d", itemCount)

	// Загружаем все заказы
	orders, err := repo.LoadAllOrders(ctx)
	if err != nil {
		t.Fatalf("LoadAllOrders failed: %v", err)
	}
	if len(orders) != 2 {
		t.Errorf("expected 2 orders, got %d", len(orders))
	}
	for _, o := range orders {
		if o.OrderUID == "load1" {
			if o.Delivery.Name != "D1" {
				t.Error("delivery not loaded for order1")
			}
			if len(o.Items) != 1 {
				t.Error("items not loaded for order1")
			}
		}
	}
}

func TestPostgresRepository_ClearAll(t *testing.T) {
	db := connectTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close test database: %v", err)
		}
	}()
	truncateTables(t, db)

	repo := &Repository{db: db}
	order := domain.Order{
		OrderUID:    "clear-test",
		TrackNumber: "T",
		Entry:       "WBIL",
		Delivery: domain.Delivery{
			Name:    "D",
			Phone:   "1",
			Zip:     "1",
			City:    "C",
			Address: "A",
			Region:  "R",
			Email:   "e@e.com",
		},
		Payment: domain.Payment{
			Transaction: "trx",
			Currency:    "USD",
			Amount:      100,
			PaymentDT:   time.Now().Unix(),
		},
		Items: []domain.Item{
			{
				ChrtID:      1,
				TrackNumber: "T",
				Price:       10,
				Name:        "I",
				TotalPrice:  10,
				NmID:        1,
			},
		},
		DateCreated: time.Now().Format(time.RFC3339),
	}
	if err := repo.SaveOrder(context.Background(), order); err != nil {
		t.Fatal(err)
	}

	if err := repo.ClearAll(context.Background()); err != nil {
		t.Fatalf("ClearAll failed: %v", err)
	}

	orders, err := repo.LoadAllOrders(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 0 {
		t.Errorf("expected 0 orders after ClearAll, got %d", len(orders))
	}
}
