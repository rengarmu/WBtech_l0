package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cache "WBtech_l0/back/cache"
	config "WBtech_l0/back/config"
	handler "WBtech_l0/back/handlers"
	consumer "WBtech_l0/back/kafka"
	service "WBtech_l0/back/service"
	Database "WBtech_l0/back/storage"
)

func main() {
	log.Println("Starting server...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	cache := cache.New()
	log.Println("Cache initialized")
	log.Printf("Connected to database %s:%s", cfg.Database.Host, cfg.Database.Port)
	db, err := Database.NewDatabaseConnect(cfg.Database)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	if err := db.Init(); err != nil {
		log.Fatal("Error initializing database:", err)
	}
	log.Println("Database initialized")

	svc := service.NewOrderService(cache, db)

	if err := svc.RestoreCache; err != nil {
		log.Fatal("Cache restore failed:", err)
	}
	log.Println("Cache restored successfully")

	kafkaConsumer := kafka.NewConsumer(svc)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kafkaConsumer.Start(ctx)
	log.Println("Kafka consumer started")

	handler := handler.NewHandler(svc)
	router := mux.NewRouter()
	router.HandleFunc("/orders", handler.GetOrders).Methods("GET")
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	shutdownComplete := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		<-sigint
		log.Println("Received shutdown signal")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Println("Error shutting down server:", err)
		}
		if err := kafkaConsumer.Close(); err != nil {
			log.Println("Error closing Kafka consumer:", err)
		}
		cancel()
		close(shutdownComplete)
	}()

	log.Println("Server started on port", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server failed:", err)
	}
	<-shutdownComplete
	log.Println("Server shutdown complete")

}
