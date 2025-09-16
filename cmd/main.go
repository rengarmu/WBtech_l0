package main

import (
	"WBtech_l0/backend"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Загружаем конфигурацию из файла config.yaml
	cfg := backend.LoadConfig("config.yaml")

	// Подключаемся к базе данных PostgreSQL
	db := backend.InitDB(cfg)
	defer db.Close()

	// Инициализируем кеш заказов (в памяти)
	cache := backend.NewOrderCache()

	// Восстанавливаем кеш из базы данных при старте
	err := backend.LoadCacheFromDB(db, cache)
	if err != nil {
		log.Fatalf("failed to load cache from DB: %v", err)
	}

	// Запускаем обработчик сообщений из Kafka в отдельной горутине
	go backend.ConsumeKafka(cfg, db, cache)

	// Регистрируем HTTP handler для получения заказа по ID (JSON API)
	http.HandleFunc("/order/", backend.MakeOrderHandler(cache, db))

	// Запускаем HTTP сервер
	addr := fmt.Sprintf("%s:%s", cfg.HTTPServer.Host, cfg.HTTPServer.Port)
	log.Printf("Starting HTTP server at %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
