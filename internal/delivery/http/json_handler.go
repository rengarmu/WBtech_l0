package httpdelivery

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"WBtech_l0/internal/domain"
	"WBtech_l0/internal/repository/cache"
	"WBtech_l0/internal/telemetry"
)

// JSONResponse стандартный формат ответа API
type JSONResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// MakeJSONOrderHandler возвращает JSON с данными заказа
func MakeJSONOrderHandler(usecase domain.OrderUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Устанавливаем заголовок для JSON ответа
		w.Header().Set("Content-Type", "application/json")

		// Извлекаем order_uid из URL
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 {
			w.WriteHeader(http.StatusBadRequest)
			// Используем форматированный вывод даже для ошибок
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			if err := enc.Encode(JSONResponse{
				Success: false,
				Error:   "order_uid required",
			}); err != nil {
				log.Printf("failed to encode error response: %v", err)
			}
			telemetry.OrdersProcessed.WithLabelValues("http", "error").Inc()
			return
		}

		orderUID := parts[3]

		// Валидация orderUID
		if !isValidOrderUID(orderUID) {
			w.WriteHeader(http.StatusBadRequest)
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			if err := enc.Encode(JSONResponse{
				Success: false,
				Error:   "Invalid order_uid format",
			}); err != nil {
				log.Printf("failed to encode error response: %v", err)
			}
			telemetry.OrdersProcessed.WithLabelValues("http", "error").Inc() //
			return
		}

		// Получаем заказ
		order, err := usecase.GetOrder(r.Context(), orderUID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			if err := enc.Encode(JSONResponse{
				Success: false,
				Error:   "Order not found",
			}); err != nil {
				log.Printf("failed to encode error response: %v", err)
			}
			telemetry.OrdersProcessed.WithLabelValues("http", "error").Inc() //
			return
		}
		telemetry.OrdersProcessed.WithLabelValues("http", "success").Inc()

		// Всегда форматированный вывод
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(JSONResponse{
			Success: true,
			Data:    order,
		}); err != nil {
			log.Printf("failed to encode success response: %v", err) // FIXED: log ignored error
		}
	}
}

// MakeJSONHealthHandler возвращает статус сервиса
func MakeJSONHealthHandler(cache *cache.OrderCache, db DBPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Проверяем подключение к БД
		dbStatus := "ok"
		if err := db.PingContext(r.Context()); err != nil {
			dbStatus = "error"
			log.Printf("Health check DB ping failed: %v", err)
		}

		// Получаем статистику кеша
		stats := cache.GetStats()

		response := map[string]interface{}{
			"status": "healthy",
			"database": map[string]string{
				"status": dbStatus,
			},
			"cache": map[string]interface{}{
				"size":   stats.Size,
				"hits":   stats.Hits,
				"misses": stats.Misses,
			},
			"timestamp": time.Now().Unix(),
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("failed to encode response: %v", err)
		}
	}
}
