// Package kafka содержит логику consumer'а для получения сообщений из Kafka
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"WBtech_l0/internal/config"
	"WBtech_l0/internal/domain"
	"WBtech_l0/internal/telemetry"

	"github.com/segmentio/kafka-go"
)

var tracer = otel.Tracer("kafka-consumer")

// ConsumeKafka подключаемся к Kafka и обрабатываем новые заказы
func ConsumeKafka(ctx context.Context, cfg config.Config, usecase domain.OrderUsecase) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{cfg.Kafka.Brokers},
		GroupID: cfg.Kafka.GroupID,
		Topic:   cfg.Kafka.Topic,
	})
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("failed to close Kafka reader: %v", err)
		}
	}()
	log.Printf("Kafka consumer started for topic: %s", cfg.Kafka.Topic)

	// Создаём writer для DLQ
	dlqWriter := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers),
		Topic:        cfg.Kafka.DLQTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}
	defer func() {
		if err := dlqWriter.Close(); err != nil {
			log.Printf("failed to close DLQ writer: %v", err)
		}
	}()
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
				time.Sleep(1 * time.Second)
				continue
			}
			// Начинаем спан для обработки сообщения
			ctx, span := tracer.Start(ctx, "process-kafka-message",
				trace.WithAttributes(
					attribute.String("message.key", string(m.Key)),
					attribute.String("topic", m.Topic),
					attribute.Int("partition", m.Partition),
					attribute.Int64("offset", m.Offset),
				))
			processStart := time.Now()
			status := "success"

			// Разбираем JSON
			var order domain.Order
			if err := json.Unmarshal(m.Value, &order); err != nil {
				log.Printf("Invalid JSON, sending to DLQ: %v", err)
				status = "error"
				span.RecordError(err)
				telemetry.OrdersProcessed.WithLabelValues("kafka", "error").Inc()
				if dlqErr := sendToDLQ(ctx, dlqWriter, m, "invalid_json", err.Error()); dlqErr != nil {
					log.Printf("Failed to send to DLQ: %v", dlqErr)
				}
				commitAndEnd(ctx, r, m, span, processStart, cfg.Kafka.Topic, status)
				continue
			}

			// Игнорируем, если нет order_uid
			if order.OrderUID == "" {
				log.Printf("Message without order_uid, sending to DLQ")
				status = "error"
				span.SetAttributes(attribute.String("error", "missing_order_uid"))
				telemetry.OrdersProcessed.WithLabelValues("kafka", "error").Inc()
				if dlqErr := sendToDLQ(ctx, dlqWriter, m, "missing_order_uid", ""); dlqErr != nil {
					log.Printf("Failed to send to DLQ: %v", dlqErr)
				}
				commitAndEnd(ctx, r, m, span, processStart, cfg.Kafka.Topic, status)
				continue
			}

			// Сохраняем заказ в транзакции
			if err = usecase.SaveOrder(ctx, order); err != nil {
				log.Printf("Failed to save order %s: %v", order.OrderUID, err)
				status = "error"
				span.RecordError(err)
				telemetry.OrdersProcessed.WithLabelValues("kafka", "error").Inc()
				if dlqErr := sendToDLQ(ctx, dlqWriter, m, "save_failed", err.Error()); dlqErr != nil {
					log.Printf("Failed to send to DLQ: %v", dlqErr)
				}
				commitAndEnd(ctx, r, m, span, processStart, cfg.Kafka.Topic, status)
				continue
			}

			// Успех
			telemetry.OrdersProcessed.WithLabelValues("kafka", "success").Inc()
			span.SetAttributes(attribute.String("order_uid", order.OrderUID))
			commitAndEnd(ctx, r, m, span, processStart, cfg.Kafka.Topic, status)
		}
	}
}

// commitAndEnd коммитит сообщение, завершает спан и записывает метрику времени
func commitAndEnd(ctx context.Context, r *kafka.Reader, m kafka.Message, span trace.Span, start time.Time, topic, status string) {
	duration := time.Since(start).Seconds()
	telemetry.KafkaMessageProcessDuration.WithLabelValues(topic, status).Observe(duration)
	span.End()
	if err := r.CommitMessages(ctx, m); err != nil {
		log.Printf("Failed to commit message: %v", err)
	}
}

// sendToDLQ отправляет сообщение в DLQ с информацией об ошибке
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
		log.Printf("Failed to marshal DLQ message: %v", err)
		return fmt.Errorf("marshal DLQ message: %w", err)
	}

	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   originalMsg.Key,
		Value: data,
		Headers: append(originalMsg.Headers,
			kafka.Header{Key: "dlq-reason", Value: []byte(reason)},
		),
	})
	if err != nil {
		return fmt.Errorf("failed to write to DLQ: %w", err)
	}
	return nil
}
