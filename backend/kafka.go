package backend

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

// ConsumeKafka подключаемся к Kafka и обрабатываем новые заказы
func ConsumeKafka(cfg Config, db *sql.DB, cache *OrderCache) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{cfg.Kafka.Brokers},
		GroupID: cfg.Kafka.GroupID,
		Topic:   cfg.Kafka.Topic,
	})
	defer r.Close()

	for {
		// FetchMessage для контроля над коммитами
		m, err := r.FetchMessage(context.Background())
		if err != nil {
			log.Printf("Kafka fetch error: %v", err)
			continue
		}

		var order Order
		if err := json.Unmarshal(m.Value, &order); err != nil {
			log.Printf("Invalid JSON, skipping: %v", err)
			// Пропускаем некорректные сообщения, но не коммитим их
			continue
		}

		// Игнорируем, если нет order_uid
		if order.OrderUID == "" {
			log.Printf("Message without order_uid, skipping")
			continue
		}

		// Сохраняем заказ в транзакции
		err = SaveOrderTx(db, order)
		if err != nil {
			log.Printf("Failed to save order %s: %v", order.OrderUID, err)
			// Не коммитим оффсет при ошибке сохранения - сообщение будет обработано повторно
			continue
		}

		// Обновляем кеш
		cache.Set(order)
		log.Printf("Order %s saved to DB", order.OrderUID)

		// Явно коммитим оффсет только после успешного сохранения в БД
		if err := r.CommitMessages(context.Background(), m); err != nil {
			log.Printf("Failed to commit message for order %s: %v", order.OrderUID, err)
		} else {
			log.Printf("Order %s processed and committed", order.OrderUID)
		}
	}
}
