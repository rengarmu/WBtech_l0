package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"time"

	"WBtech_l0/internal/config"
	"WBtech_l0/internal/domain"
	"WBtech_l0/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func main() {
	// Парсим аргументы командной строки
	var (
		configPath    string
		numOrders     int
		sendToKafka   bool
		clearExisting bool
	)

	flag.StringVar(&configPath, "config", "configs/config.yaml", "path to config file")
	flag.IntVar(&numOrders, "count", 10, "number of test orders to create")
	flag.BoolVar(&sendToKafka, "kafka", true, "send orders to Kafka")
	flag.BoolVar(&clearExisting, "clear", false, "clear existing data before seeding")
	flag.Parse()

	// Загружаем конфигурацию
	cfg := config.LoadConfig(configPath)
	log.Printf("Config loaded from: %s", configPath)

	// Подключаемся к базе данных
	db := postgres.InitDB(*cfg)
	defer func() {
		log.Println("Closing database connection...")
		db.Close()
	}()

	// Опционально очищаем существующие данные
	if clearExisting {
		log.Println("Clearing existing data...")
		if err := clearDatabase(db); err != nil {
			log.Fatalf("Failed to clear database: %v", err)
		}
		log.Println("Database cleared")
	}

	// Создаем тестовые заказы
	log.Printf("Creating %d test orders...", numOrders)
	orders := make([]domain.Order, 0, numOrders)

	for i := 0; i < numOrders; i++ {
		order := createTestOrder(i)
		orders = append(orders, order)

		// Сохраняем в БД
		err := postgres.SaveOrderTx(db, order)
		if err != nil {
			log.Printf("Failed to save order %s: %v", order.OrderUID, err)
			continue
		}

		log.Printf("Order %d saved to DB: %s", i+1, order.OrderUID)
	}

	log.Printf("Successfully saved %d orders to database", len(orders))

	// Опционально отправляем в Kafka
	if sendToKafka && len(orders) > 0 {
		log.Println("Sending orders to Kafka...")
		if err := sendOrdersToKafka(cfg, orders); err != nil {
			log.Fatalf("Failed to send orders to Kafka: %v", err)
		}
		log.Printf("Successfully sent %d orders to Kafka topic: %s", len(orders), cfg.Kafka.Topic)
	}

	log.Println("Seed completed successfully!")
}

// createTestOrder создает тестовый заказ с уникальными данными
func createTestOrder(index int) domain.Order {
	// Генерируем уникальный ID
	orderUID := uuid.New().String()[:8] + "-test-" + time.Now().Format("150405")

	return domain.Order{
		OrderUID:    orderUID,
		TrackNumber: "WB-TEST-" + time.Now().Format("20060102") + "-" + uuid.New().String()[:6],
		Entry:       "WBIL",
		Delivery: domain.Delivery{
			Name:    "Тестовый Пользователь",
			Phone:   "+7" + generatePhoneNumber(index),
			Zip:     "123" + generateDigits(3),
			City:    "Москва",
			Address: "ул. Тестовая, д. " + generateDigits(2) + ", кв. " + generateDigits(2),
			Region:  "Московская область",
			Email:   "test" + generateDigits(3) + "@example.com",
		},
		Payment: domain.Payment{
			Transaction:  "TRX-" + uuid.New().String()[:12],
			RequestID:    "",
			Currency:     "RUB",
			Provider:     "test-provider",
			Amount:       1000 + (index * 500),
			PaymentDT:    time.Now().Unix(),
			Bank:         "Тест Банк",
			DeliveryCost: 150,
			GoodsTotal:   850 + (index * 500),
			CustomFee:    0,
		},
		Items: []domain.Item{
			{
				ChrtID:      100000 + index,
				TrackNumber: "WB-TEST-ITEM-1",
				Price:       500,
				Rid:         "RID-" + uuid.New().String()[:8],
				Name:        "Тестовый товар 1",
				Sale:        0,
				Size:        "M",
				TotalPrice:  500,
				NmID:        1000 + index,
				Brand:       "Тест Бренд",
				Status:      200,
			},
			{
				ChrtID:      200000 + index,
				TrackNumber: "WB-TEST-ITEM-2",
				Price:       350,
				Rid:         "RID-" + uuid.New().String()[:8],
				Name:        "Тестовый товар 2",
				Sale:        10,
				Size:        "L",
				TotalPrice:  315,
				NmID:        2000 + index,
				Brand:       "Тест Бренд",
				Status:      100,
			},
		},
		Locale:            "ru",
		InternalSignature: "",
		CustomerID:        "cust-" + generateDigits(5),
		DeliveryService:   "test-delivery",
		Shardkey:          generateDigits(1),
		SmID:              1,
		DateCreated:       time.Now().Format(time.RFC3339),
		OofShard:          generateDigits(1),
	}
}

// generatePhoneNumber генерирует тестовый номер телефона
func generatePhoneNumber(index int) string {
	return "900" + generateDigits(7)
}

// generateDigits генерирует строку из n цифр
func generateDigits(n int) string {
	digits := "0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = digits[time.Now().UnixNano()%10]
		time.Sleep(1) // чтобы избежать одинаковых чисел
	}
	return string(result)
}

// clearDatabase очищает все таблицы
func clearDatabase(db *sql.DB) error {
	tables := []string{"items", "payments", "deliveries", "orders"}

	for _, table := range tables {
		if _, err := db.Exec("DELETE FROM " + table); err != nil {
			return err
		}
	}

	// Сбрасываем последовательности для PostgreSQL
	if _, err := db.Exec("ALTER SEQUENCE items_id_seq RESTART WITH 1"); err != nil {
		// Игнорируем ошибку, если последовательности нет
	}

	return nil
}

// sendOrdersToKafka отправляет заказы в Kafka
func sendOrdersToKafka(cfg *config.Config, orders []domain.Order) error {
	// Настройка Kafka writer
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers),
		Topic:        cfg.Kafka.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 100 * time.Millisecond,
		BatchSize:    100,
		RequiredAcks: kafka.RequireAll,
	}
	defer w.Close()

	// Создаем сообщения
	messages := make([]kafka.Message, 0, len(orders))
	for _, order := range orders {
		value, err := json.Marshal(order)
		if err != nil {
			return err
		}

		messages = append(messages, kafka.Message{
			Key:   []byte(order.OrderUID),
			Value: value,
			Headers: []kafka.Header{
				{Key: "source", Value: []byte("seed")},
				{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
			},
		})
	}
	// Проверка на пустые сообщения
	if len(messages) == 0 {
		return nil
	}

	// Отправляем сообщения
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return w.WriteMessages(ctx, messages...)
}
