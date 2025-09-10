package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"wb_tech/back/models"

	"github.com/segmentio/kafka-go"
)

// Consumer обертка для Kafka reader
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer создает новый Kafka consumer
func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,          // Список брокеров Kafka
		Topic:    topic,            // Топик для подписки
		GroupID:  groupID,          // Group ID для consumer group
		MinBytes: 10e3,             // Минимальный размер батча (10KB)
		MaxBytes: 10e6,             // Максимальный размер батча (10MB)
		MaxWait:  10 * time.Second, // Максимальное время ожидания сообщений
	})

	return &Consumer{reader: reader}
}

// Consume запускает бесконечный цикл потребления сообщений, обрабатывает сообщения и передает их в processFunc

func (c *Consumer) Consume(ctx context.Context, processFunc func(order models.Order) error) error {
	for {
		select {
		case <-ctx.Done():
			return c.reader.Close()
		default:
			// Чтение сообщения из Kafka с таймаутом
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			// Десериализация JSON сообщения в структуру Order
			var order models.Order
			if err := json.Unmarshal(msg.Value, &order); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			// Обработка заказа через предоставленную функцию
			if err := processFunc(order); err != nil {
				log.Printf("Error processing order: %v", err)
			} else {
				log.Printf("Successfully processed order: %s", order.OrderUID)
			}
		}
	}
}

// Close закрывает Kafka reader, освобождает ресурсы и завершает соединения
func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("failed to close Kafka reader: %w", err)
	}
	return nil
}
