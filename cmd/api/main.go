package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"WBtech_l0/internal/config"
	httpdelivery "WBtech_l0/internal/delivery/http"
	"WBtech_l0/internal/repository/cache"
	"WBtech_l0/internal/repository/postgres"
	"WBtech_l0/internal/usecase/kafka"
)

func main() {
	// Загружаем конфигурацию
	configPath := filepath.Join("configs", "config.yaml")
	cfg := config.LoadConfig(configPath)
	log.Printf("Config loaded from: %s", configPath)

	// Подключаемся к базе данных
	db := postgres.InitDB(*cfg)
	defer db.Close()

	// Инициализируем кеш
	orderCache := cache.NewOrderCache()
	orderCache.SetDefaultTTL(1 * time.Hour) // Устанавливаем TTL 1 час
	orderCache.SetMaxSize(1000)             // Максимум 5000 записей

	// Восстанавливаем кеш из БД
	log.Println("Loading cache from database...")
	err := postgres.LoadCacheFromDB(db, orderCache)
	if err != nil {
		log.Fatalf("failed to load cache from DB: %v", err)
	}
	log.Printf("Cache loaded with %d orders", len(orderCache.GetAll()))

	// Создаем контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Канал для сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем Kafka consumer
	go func() {
		kafka.ConsumeKafka(ctx, *cfg, db, orderCache)
	}()
	log.Println("Kafka consumer started")

	// Создаем и запускаем сервер
	server := httpdelivery.NewServer(cfg, orderCache, db)

	// Запускаем сервер в горутине
	go func() {
		if err := server.Run(); err != nil {
			log.Printf("Server error: %v", err)
			cancel()
		}
	}()

	// Ждем сигнал завершения
	<-sigChan
	log.Println("Shutting down gracefully...")

	// Даем время на завершение операций
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Останавливаем HTTP сервер
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	// Останавливаем Kafka consumer
	cancel()

	// Даем время на завершение Kafka consumer
	time.Sleep(2 * time.Second)

	log.Println("Shutdown complete")
}
