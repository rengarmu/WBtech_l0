package httpdelivery

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time" // ИЗМЕНЕНО: добавлен импорт time для health check

	"WBtech_l0/internal/repository/cache"
)

// ИЗМЕНЕНО: Добавлена структура для стандартного JSON ответа
// JSONResponse стандартный формат ответа API
type JSONResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// MakeJSONOrderHandler возвращает JSON с данными заказа
func MakeJSONOrderHandler(cache *cache.OrderCache, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Устанавливаем заголовок для JSON ответа
		w.Header().Set("Content-Type", "application/json")

		// Извлекаем order_uid из URL
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 {
			w.WriteHeader(http.StatusBadRequest)
			// ИЗМЕНЕНО: используем форматированный вывод даже для ошибок
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			enc.Encode(JSONResponse{
				Success: false,
				Error:   "order_uid required",
			})
			return
		}
		orderUID := parts[3]

		// Валидация orderUID
		if !isValidOrderUID(orderUID) {
			w.WriteHeader(http.StatusBadRequest)
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			enc.Encode(JSONResponse{
				Success: false,
				Error:   "Invalid order_uid format",
			})
			return
		}

		// Получаем заказ
		order, found := getOrder(orderUID, cache, db)
		if !found {
			w.WriteHeader(http.StatusNotFound)
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			enc.Encode(JSONResponse{
				Success: false,
				Error:   "Order not found",
			})
			return
		}

		// ИЗМЕНЕНО: всегда форматированный вывод
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(JSONResponse{
			Success: true,
			Data:    order,
		})
	}
}

// MakeJSONHealthHandler возвращает статус сервиса
func MakeJSONHealthHandler(cache *cache.OrderCache, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Проверяем подключение к БД
		dbStatus := "ok"
		if err := db.Ping(); err != nil {
			dbStatus = "error"
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

		json.NewEncoder(w).Encode(response)
	}
}
