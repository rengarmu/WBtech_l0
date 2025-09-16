package backend

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// MakeOrderHandler — HTTP обработчик для получения заказа по ID
func MakeOrderHandler(cache *OrderCache, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем order_uid из URL
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 {
			http.Error(w, "order_uid required", http.StatusBadRequest)
			return
		}
		orderUID := parts[2]

		// Пробуем достать заказ из кеша
		if order, ok := cache.Get(orderUID); ok {
			json.NewEncoder(w).Encode(order)
			return
		}

		// Если в кеше нет — достаем из базы
		order, err := GetOrderFromDB(db, orderUID)
		if err != nil {
			log.Printf("error loading order %s from DB: %v", orderUID, err)
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}

		// Сохраняем в кеш для ускорения следующих запросов
		cache.Set(order)

		json.NewEncoder(w).Encode(order)
	}
}
