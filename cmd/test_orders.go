package main

import (
	"WBtech_l0/backend"
	"log"
	"time"
)

func main() {
	// Загружаем конфигурацию из файла config.yaml
	cfg := backend.LoadConfig("config.yaml")

	// Подключаемся к базе данных PostgreSQL
	db := backend.InitDB(cfg)
	defer db.Close()

	// Создаем тестовые заказы
	testOrders := generateTestOrders()

	// Сохраняем каждый заказ в БД
	for _, order := range testOrders {
		err := backend.SaveOrderTx(db, order)
		if err != nil {
			log.Printf("Failed to save order %s: %v", order.OrderUID, err)
		} else {
			log.Printf("Successfully saved order: %s", order.OrderUID)
		}
	}

	log.Println("Finished seeding test orders")
}

func generateTestOrders() []backend.Order {
	now := time.Now().Format(time.RFC3339)

	return []backend.Order{
		{
			OrderUID:          "test-order-001",
			TrackNumber:       "TN001TEST",
			Entry:             "WBIL",
			Locale:            "en",
			InternalSignature: "",
			CustomerID:        "test-customer-1",
			DeliveryService:   "meest",
			Shardkey:          "9",
			SmID:              99,
			DateCreated:       now,
			OofShard:          "1",
			Delivery: backend.Delivery{
				Name:    "John Doe",
				Phone:   "+1234567890",
				Zip:     "123456",
				City:    "New York",
				Address: "123 Main St",
				Region:  "NY",
				Email:   "john.doe@example.com",
			},
			Payment: backend.Payment{
				Transaction:  "txn-001-test",
				RequestID:    "",
				Currency:     "USD",
				Provider:     "wbpay",
				Amount:       2000,
				PaymentDT:    time.Now().Unix(),
				Bank:         "alpha",
				DeliveryCost: 500,
				GoodsTotal:   1500,
				CustomFee:    0,
			},
			Items: []backend.Item{
				{
					ChrtID:      1234567,
					TrackNumber: "TN001TEST",
					Price:       1000,
					Rid:         "rid-001-test",
					Name:        "Test Product 1",
					Sale:        0,
					Size:        "M",
					TotalPrice:  1000,
					NmID:        1111111,
					Brand:       "Test Brand",
					Status:      202,
				},
			},
		},
		{
			OrderUID:          "test-order-002",
			TrackNumber:       "TN002TEST",
			Entry:             "WBIL",
			Locale:            "ru",
			InternalSignature: "internal-sig-2",
			CustomerID:        "test-customer-2",
			DeliveryService:   "russian-post",
			Shardkey:          "8",
			SmID:              88,
			DateCreated:       now,
			OofShard:          "2",
			Delivery: backend.Delivery{
				Name:    "Иван Иванов",
				Phone:   "+79161234567",
				Zip:     "101000",
				City:    "Москва",
				Address: "ул. Тверская, д. 1",
				Region:  "Москва",
				Email:   "ivan.ivanov@example.ru",
			},
			Payment: backend.Payment{
				Transaction:  "txn-002-test",
				RequestID:    "req-002",
				Currency:     "RUB",
				Provider:     "sberpay",
				Amount:       5000,
				PaymentDT:    time.Now().Unix(),
				Bank:         "sber",
				DeliveryCost: 300,
				GoodsTotal:   4700,
				CustomFee:    0,
			},
			Items: []backend.Item{
				{
					ChrtID:      7654321,
					TrackNumber: "TN002TEST",
					Price:       2500,
					Rid:         "rid-002-test",
					Name:        "Тестовый товар 2",
					Sale:        10,
					Size:        "L",
					TotalPrice:  2250,
					NmID:        2222222,
					Brand:       "Тестовый бренд",
					Status:      202,
				},
				{
					ChrtID:      7654322,
					TrackNumber: "TN002TEST-2",
					Price:       2500,
					Rid:         "rid-002-test-2",
					Name:        "Дополнительный товар",
					Sale:        0,
					Size:        "S",
					TotalPrice:  2500,
					NmID:        2222223,
					Brand:       "Другой бренд",
					Status:      202,
				},
			},
		},
		{
			OrderUID:          "b563feb7b2b84b6test",
			TrackNumber:       "WBILMTESTTRACK",
			Entry:             "WBIL",
			Locale:            "en",
			InternalSignature: "",
			CustomerID:        "test",
			DeliveryService:   "meest",
			Shardkey:          "9",
			SmID:              99,
			DateCreated:       "2021-11-26T06:22:19Z",
			OofShard:          "1",
			Delivery: backend.Delivery{
				Name:    "Test Testov",
				Phone:   "+9720000000",
				Zip:     "2639809",
				City:    "Kiryat Mozkin",
				Address: "Ploshad Mira 15",
				Region:  "Kraiot",
				Email:   "test@gmail.com",
			},
			Payment: backend.Payment{
				Transaction:  "b563feb7b2b84b6test",
				RequestID:    "",
				Currency:     "USD",
				Provider:     "wbpay",
				Amount:       1817,
				PaymentDT:    1637907727,
				Bank:         "alpha",
				DeliveryCost: 1500,
				GoodsTotal:   317,
				CustomFee:    0,
			},
			Items: []backend.Item{
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
		},
		{
			OrderUID:          "test-order-empty-items",
			TrackNumber:       "TN003TEST",
			Entry:             "WBIL",
			Locale:            "en",
			InternalSignature: "",
			CustomerID:        "test-customer-3",
			DeliveryService:   "dhl",
			Shardkey:          "7",
			SmID:              77,
			DateCreated:       now,
			OofShard:          "3",
			Delivery: backend.Delivery{
				Name:    "Alice Smith",
				Phone:   "+441234567890",
				Zip:     "SW1A 1AA",
				City:    "London",
				Address: "10 Downing Street",
				Region:  "England",
				Email:   "alice.smith@example.uk",
			},
			Payment: backend.Payment{
				Transaction:  "txn-003-test",
				RequestID:    "req-003",
				Currency:     "GBP",
				Provider:     "stripe",
				Amount:       7500,
				PaymentDT:    time.Now().Unix(),
				Bank:         "barclays",
				DeliveryCost: 1000,
				GoodsTotal:   6500,
				CustomFee:    0,
			},
			Items: []backend.Item{}, // Пустой список товаров
		},
		{
			OrderUID:          "test-order-multiple-items",
			TrackNumber:       "TN004TEST",
			Entry:             "WBIL",
			Locale:            "de",
			InternalSignature: "internal-sig-4",
			CustomerID:        "test-customer-4",
			DeliveryService:   "dpd",
			Shardkey:          "6",
			SmID:              66,
			DateCreated:       now,
			OofShard:          "4",
			Delivery: backend.Delivery{
				Name:    "Hans Müller",
				Phone:   "+49123456789",
				Zip:     "10115",
				City:    "Berlin",
				Address: "Unter den Linden 1",
				Region:  "Berlin",
				Email:   "hans.muller@example.de",
			},
			Payment: backend.Payment{
				Transaction:  "txn-004-test",
				RequestID:    "req-004",
				Currency:     "EUR",
				Provider:     "paypal",
				Amount:       12000,
				PaymentDT:    time.Now().Unix(),
				Bank:         "deutsche",
				DeliveryCost: 700,
				GoodsTotal:   11300,
				CustomFee:    0,
			},
			Items: []backend.Item{
				{
					ChrtID:      1111111,
					TrackNumber: "TN004TEST-1",
					Price:       4000,
					Rid:         "rid-004-test-1",
					Name:        "Produkt 1",
					Sale:        15,
					Size:        "XL",
					TotalPrice:  3400,
					NmID:        3333333,
					Brand:       "Deutsche Marke",
					Status:      202,
				},
				{
					ChrtID:      1111112,
					TrackNumber: "TN004TEST-2",
					Price:       5000,
					Rid:         "rid-004-test-2",
					Name:        "Produkt 2",
					Sale:        20,
					Size:        "M",
					TotalPrice:  4000,
					NmID:        3333334,
					Brand:       "Andere Marke",
					Status:      202,
				},
				{
					ChrtID:      1111113,
					TrackNumber: "TN004TEST-3",
					Price:       4500,
					Rid:         "rid-004-test-3",
					Name:        "Produkt 3",
					Sale:        10,
					Size:        "S",
					TotalPrice:  4050,
					NmID:        3333335,
					Brand:       "Dritte Marke",
					Status:      202,
				},
			},
		},
	}
}
