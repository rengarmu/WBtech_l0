package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"WBtech_l0/internal/config"
	"WBtech_l0/internal/domain"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func main() {
	var (
		msgType    = flag.String("type", "valid", "Type of message: valid or invalid")
		count      = flag.Int("count", 1, "Number of messages to send")
		interval   = flag.Duration("interval", 1*time.Second, "Interval between messages")
		configPath = flag.String("config", "configs/config.yaml", "Path to config file")
	)
	flag.Parse()

	cfg := config.LoadConfig(*configPath)

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{cfg.Kafka.Brokers},
		Topic:        cfg.Kafka.Topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: int(kafka.RequireOne),
		Async:        false,
	})
	defer writer.Close()

	log.Printf("Producer started, sending %d %s message(s) to topic %s every %v",
		*count, *msgType, cfg.Kafka.Topic, *interval)

	for i := 0; i < *count; i++ {
		// Генерируем сообщение и отдельно получаем orderUID для ключа
		msgData, orderUID, err := generateMessage(*msgType)
		if err != nil {
			log.Fatalf("Failed to generate message: %v", err)
		}

		// Определяем ключ (для невалидных сообщений может быть пустым)
		var key []byte
		if orderUID != "" {
			key = []byte(orderUID)
		}

		// Отправляем
		err = writer.WriteMessages(context.Background(), kafka.Message{
			Key:   key,
			Value: msgData,
		})
		if err != nil {
			log.Printf("Failed to send message %d: %v", i+1, err)
		} else {
			log.Printf("Message %d sent: order_uid=%s", i+1, orderUID)
		}

		time.Sleep(*interval)
	}
}

// generateMessage создаёт сообщение заданного типа и возвращает JSON-данные и orderUID.
func generateMessage(msgType string) (data []byte, orderUID string, err error) {
	var order domain.Order

	switch msgType {
	case "valid":
		order = loadValidOrderTemplate()
		orderUID = uuid.New().String()
		order.OrderUID = orderUID
		order.Payment.Transaction = orderUID
	case "invalid":
		order = domain.Order{
			TrackNumber: "INVALID-TRACK",
			Entry:       "INVALID",
			Delivery: domain.Delivery{
				Name:    "Test",
				Phone:   "123",
				City:    "City",
				Address: "Addr",
			},
			Payment: domain.Payment{
				Currency:  "USD",
				Amount:    100,
				PaymentDT: time.Now().Unix(),
			},
			Items: []domain.Item{
				{
					ChrtID:      1,
					TrackNumber: "INVALID-TRACK",
					Price:       10,
					Name:        "Test Item",
					TotalPrice:  10,
					NmID:        1,
				},
			},
		}
		orderUID = "" // невалидный заказ не имеет UID
	default:
		return nil, "", fmt.Errorf("unknown message type: %s", msgType)
	}

	data, err = json.Marshal(order)
	if err != nil {
		return nil, "", fmt.Errorf("marshal order: %w", err)
	}
	return data, orderUID, nil
}

// loadValidOrderTemplate загружает пример валидного заказа из файла model.json
func loadValidOrderTemplate() domain.Order {
	path := filepath.Join("model.json")
	file, err := os.Open(path)
	if err == nil {
		defer file.Close()
		var order domain.Order
		if err := json.NewDecoder(file).Decode(&order); err == nil {
			return order
		}
		log.Printf("Warning: cannot decode model.json: %v", err)
	}
	log.Printf("Using hardcoded order template")
	return hardcodedValidOrder()
}

// hardcodedValidOrder возвращает жёстко заданный валидный заказ (резервный вариант)
func hardcodedValidOrder() domain.Order {
	return domain.Order{
		OrderUID:    "placeholder",
		TrackNumber: "WBILMTESTTRACK",
		Entry:       "WBIL",
		Delivery: domain.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: domain.Payment{
			Transaction:  "placeholder",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDT:    1637907727,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
		},
		Items: []domain.Item{
			{
				ChrtID:      9934930,
				TrackNumber: "WBILMTESTTRACK",
				Price:       453,
				Rid:         "ab4219087a764ae0btest",
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NmID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
		Locale:          "en",
		CustomerID:      "test",
		DeliveryService: "meest",
		Shardkey:        "9",
		SmID:            99,
		DateCreated:     "2021-11-26T06:22:19Z",
		OofShard:        "1",
	}
}
