package backend

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

// ConsumeKafka — подписываемся на Kafka и обрабатываем новые заказы
func ConsumeKafka(cfg Config, db *sql.DB, cache *OrderCache) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{cfg.Kafka.Brokers},
		GroupID: cfg.Kafka.GroupID,
		Topic:   cfg.Kafka.Topic,
	})
	defer r.Close()

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Kafka read error: %v", err)
			continue
		}

		var order Order
		if err := json.Unmarshal(m.Value, &order); err != nil {
			log.Printf("Invalid JSON, skipping: %v", err)
			continue
		}

		// Игнорируем, если нет order_uid
		if order.OrderUID == "" {
			log.Printf("Message without order_uid, skipping")
			continue
		}

		// Сохраняем заказ в транзакции (атомарно)
		err = SaveOrderTx(db, order)
		if err != nil {
			log.Printf("Failed to save order %s: %v", order.OrderUID, err)
			continue
		}

		// Обновляем кеш
		cache.Set(order)
		log.Printf("Order %s processed and cached", order.OrderUID)
	}
}
