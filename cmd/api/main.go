// Package main - основной сервис API для обработки заказов
// Запускает HTTP-сервер, Kafka consumer и инициализирует зависимости
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"WBtech_l0/internal/config"
	httpdelivery "WBtech_l0/internal/delivery/http"
	"WBtech_l0/internal/repository/cache"
	"WBtech_l0/internal/repository/postgres"
	"WBtech_l0/internal/telemetry"
	"WBtech_l0/internal/usecase"
	"WBtech_l0/internal/usecase/kafka"
)

func main() {
	// Загружаем конфигурацию
	configPath := filepath.Join("configs", "config.yaml")
	cfg := config.LoadConfig(configPath)
	log.Printf("Config loaded from: %s", configPath)

	// Запускаем миграции
	if err := runMigrations(*cfg); err != nil {
		log.Fatalf("Migration error: %v", err)
	}

	// Создаем контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Инициализация трейсера (Jaeger). URL можно вынести в конфиг
	shutdownTracer := telemetry.InitTracer("order-service", cfg.Telemetry.OTLPEndpoint)
	defer shutdownTracer()

	// Репозиторий
	repo := postgres.InitDB(*cfg)
	defer func() {
		if err := repo.Close(); err != nil {
			log.Printf("failed to close repository: %v", err)
		}
	}()

	// Инициализируем кеш
	orderCache := cache.NewOrderCache(cfg.Cache.DefaultTTL, cfg.Cache.MaxSize)

	// Восстанавливаем кеш из БД
	log.Println("Loading cache from database...")
	if err := postgres.LoadCacheFromDB(ctx, repo, orderCache); err != nil {
		log.Printf("failed to load cache from DB: %v", err)
		// Закрываем ресурсы вручную
		if closeErr := repo.Close(); closeErr != nil {
			log.Printf("error closing repository: %v", closeErr)
		}
		shutdownTracer()
		cancel()
		log.Fatal("exiting due to cache load error") //nolint:gocritic
	}

	log.Printf("Cache loaded with %d orders", len(orderCache.GetAll()))
	// Usecase
	orderUsecase := usecase.NewOrderUsecase(repo, orderCache)

	// Канал для сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем Kafka consumer
	go func() {
		kafka.ConsumeKafka(ctx, *cfg, orderUsecase)
	}()
	log.Println("Kafka consumer started")

	// Создаем и запускаем сервер
	server := httpdelivery.NewServer(cfg, orderUsecase, repo, orderCache)

	// Добавляем отдельный HTTP-маршрут для метрик (можно на другом порту или на основном)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		addr := ":" + cfg.Telemetry.MetricsPort
		if err := http.ListenAndServe(addr, nil); err != nil { // стандартный порт для метрик
			log.Printf("Metrics server error: %v", err)
		}
	}()

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

func runMigrations(cfg config.Config) error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.Database)
	m, err := migrate.New("file://"+cfg.MigrationsPath, dsn)

	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	defer func() {
		if sourceErr, dbErr := m.Close(); sourceErr != nil || dbErr != nil {
			log.Printf("failed to close migrator: sourceErr=%v, dbErr=%v", sourceErr, dbErr)
		}
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Println("Migrations applied successfully")
	return nil
}
