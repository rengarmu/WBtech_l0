package kafka

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"WBtech_l0/internal/config"
	"WBtech_l0/internal/domain"
	"WBtech_l0/internal/repository/cache"
	"WBtech_l0/internal/repository/postgres"

	"github.com/segmentio/kafka-go"
)

// ConsumeKafka подключаемся к Kafka и обрабатываем новые заказы
func ConsumeKafka(ctx context.Context, cfg config.Config, db *sql.DB, cache *cache.OrderCache) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{cfg.Kafka.Brokers},
		GroupID: cfg.Kafka.GroupID,
		Topic:   cfg.Kafka.Topic,
	})
	defer r.Close()
	log.Printf("Kafka consumer started for topic: %s", cfg.Kafka.Topic)

	// Создаём writer для DLQ
	dlqWriter := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers),
		Topic:        cfg.Kafka.DLQTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}
	defer dlqWriter.Close()
	log.Printf("DLQ topic: %s", cfg.Kafka.DLQTopic)

	for {
		select {
		case <-ctx.Done():
			log.Println("Kafka consumer stopped")
			return
		default:
			// FetchMessage для контроля над коммитами
			m, err := r.FetchMessage(ctx)
			if err != nil {
				log.Printf("Kafka fetch error: %v", err)
				time.Sleep(1 * time.Second) // Небольшая задержка при ошибке
				continue
			}

			var order domain.Order
			if err := json.Unmarshal(m.Value, &order); err != nil {
				log.Printf("Invalid JSON, skipping: %v", err)
				// Некорректные сообщения отправляем в DLQ
				if dlqErr := sendToDLQ(ctx, dlqWriter, m, "invalid_json", err.Error()); dlqErr != nil {
					log.Printf("Failed to send to DLQ: %v", dlqErr)
				}
				// Коммитим, чтобы не застревать на этом сообщении
				if commitErr := r.CommitMessages(ctx, m); commitErr != nil {
					log.Printf("Failed to commit message after DLQ send: %v", commitErr)
				}

				continue
			}

			// Игнорируем, если нет order_uid
			if order.OrderUID == "" {
				log.Printf("Message without order_uid, sending to DLQ")
				// Отправляем в DLQ
				if dlqErr := sendToDLQ(ctx, dlqWriter, m, "missing_order_uid", ""); dlqErr != nil {
					log.Printf("Failed to send to DLQ: %v", dlqErr)
				}
				if commitErr := r.CommitMessages(ctx, m); commitErr != nil {
					log.Printf("Failed to commit message after DLQ send: %v", commitErr)
				}
				continue
			}

			// Сохраняем заказ в транзакции
			err = postgres.SaveOrderTx(db, order)
			if err != nil {
				log.Printf("Failed to save order %s: %v", order.OrderUID, err)
				// Отправляем в DLQ
				if dlqErr := sendToDLQ(ctx, dlqWriter, m, "save_failed", err.Error()); dlqErr != nil {
					log.Printf("Failed to send to DLQ: %v", dlqErr)
				}
				if commitErr := r.CommitMessages(ctx, m); commitErr != nil {
					log.Printf("Failed to commit message after DLQ send: %v", commitErr)
				}
				continue
			}

			// Обновляем кеш
			cache.Set(order)
			log.Printf("Order %s saved to DB", order.OrderUID)

			// Явно коммитим оффсет только после успешного сохранения в БД
			if err := r.CommitMessages(ctx, m); err != nil {
				log.Printf("Failed to commit message for order %s: %v", order.OrderUID, err)
			} else {
				log.Printf("Order %s processed and committed", order.OrderUID)
			}
		}
	}
}

// Вспомогательная функция для отправки в DLQ
func sendToDLQ(ctx context.Context, writer *kafka.Writer, originalMsg kafka.Message, reason string, details string) error {
	dlqMsg := struct {
		OriginalMessage json.RawMessage `json:"original_message"`
		Reason          string          `json:"reason"`
		Details         string          `json:"details"`
		Timestamp       int64           `json:"timestamp"`
	}{
		OriginalMessage: originalMsg.Value,
		Reason:          reason,
		Details:         details,
		Timestamp:       time.Now().Unix(),
	}

	data, err := json.Marshal(dlqMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ message: %w", err)
	}

	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   originalMsg.Key,
		Value: data,
		Headers: append(originalMsg.Headers,
			kafka.Header{Key: "dlq-reason", Value: []byte(reason)},
		),
	})
	if err != nil {
		return fmt.Errorf("failed to write to DLQ: %w", err) // ИСПРАВЛЕНО: обёрнутая ошибка
	}
	return nil
}
