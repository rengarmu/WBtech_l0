package httpdelivery

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"WBtech_l0/internal/domain"
	"WBtech_l0/internal/repository/cache"
	"WBtech_l0/internal/repository/postgres"
)

// OrderHandlerData содержит данные для шаблона
type OrderHandlerData struct {
	Order domain.Order
	Found bool
}

func MakeOrderHandler(cache *cache.OrderCache, db *sql.DB) http.HandlerFunc {
	// Предзагружаем шаблон
	tmpl, err := loadTemplate()
	if err != nil {
		log.Printf("Warning: could not preload template: %v", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 {
			renderOrderNotFound(w, "")
			return
		}
		orderUID := parts[2]

		// Валидация orderUID
		if !isValidOrderUID(orderUID) {
			http.Error(w, "Invalid order_uid format", http.StatusBadRequest)
			return
		}

		// Получаем заказ
		order, found := getOrder(orderUID, cache, db)
		if !found {
			renderOrderNotFound(w, orderUID)
			return
		}

		// Рендерим шаблон
		renderOrderTemplate(w, tmpl, order, found)
	}
}

// getOrder получает заказ из кеша или БД
func getOrder(orderUID string, cache *cache.OrderCache, db *sql.DB) (domain.Order, bool) {
	// Пробуем из кеша
	if cachedOrder, ok := cache.Get(orderUID); ok {
		log.Printf("Order %s found in cache", orderUID)
		return cachedOrder, true
	}

	// Пробуем из БД
	order, err := postgres.GetOrderFromDB(db, orderUID)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("Error loading order %s from DB: %v", orderUID, err)
		}
		return domain.Order{}, false
	}

	// Сохраняем в кеш
	cache.Set(order)
	log.Printf("Order %s loaded from DB and cached", orderUID)
	return order, true
}

// renderOrderTemplate рендерит шаблон с данными заказа
func renderOrderTemplate(w http.ResponseWriter, tmpl *template.Template, order domain.Order, found bool) {
	if tmpl == nil {
		var err error
		tmpl, err = loadTemplate()
		if err != nil {
			log.Printf("Error loading template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	data := OrderHandlerData{
		Order: order,
		Found: found,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// isValidOrderUID проверяет, что orderUID не пустой и имеет допустимую длину
func isValidOrderUID(orderUID string) bool {
	if orderUID == "" || utf8.RuneCountInString(orderUID) > 255 {
		return false
	}

	for _, r := range orderUID {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}

	return true
}

// templateFunctions - функции для использования в шаблоне
var templateFunctions = template.FuncMap{
	"formatCurrency": func(amount interface{}) string {
		if amount == nil {
			return "Не указана"
		}

		switch v := amount.(type) {
		case int:
			if v == 0 {
				return "0 руб."
			}
			return fmt.Sprintf("%d руб.", v)
		case float64:
			if v == 0 {
				return "0 руб."
			}
			return fmt.Sprintf("%.2f руб.", v)
		default:
			return fmt.Sprintf("%v руб.", amount)
		}
	},
	"formatDate": func(dateString string) string {
		if dateString == "" {
			return "Не указана"
		}
		// Простая форматировка даты
		return strings.Replace(dateString, "T", " ", 1)
	},
	"formatPaymentDate": func(paymentDT int64) string {
		if paymentDT == 0 {
			return "Не указана"
		}
		// Преобразуем Unix timestamp в дату
		return time.Unix(paymentDT, 0).Format("2006-01-02 15:04:05")
	},
	"getStatusClass": func(status int) string {
		if status >= 200 {
			return "delivered"
		}
		if status >= 100 {
			return "pending"
		}
		return "cancelled"
	},
	"getStatusText": func(status int) string {
		if status >= 200 {
			return "Доставлен"
		}
		if status >= 100 {
			return "В обработке"
		}
		return "Отменен"
	},
}

// getTemplatePath возвращает правильный путь к шаблону
func getTemplatePath() (string, error) {
	// Пробуем несколько возможных путей
	possiblePaths := []string{
		"web/order_template.html", //запускаем из корня проекта
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("template file not found in any of the expected locations")
}

// loadTemplate загружает HTML-шаблон
func loadTemplate() (*template.Template, error) {
	templatePath, err := getTemplatePath()
	if err != nil {
		return nil, err
	}

	// Получаем абсолютный путь для отладки
	absPath, _ := filepath.Abs(templatePath)
	log.Printf("Loading template from: %s", absPath)

	tmpl, err := template.New("order_template.html").Funcs(templateFunctions).ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %v", err)
	}

	return tmpl, nil
}

// MakeOrderHandler — HTTP обработчик для получения заказа по ID с HTML-рендерингом
// func MakeOrderHandler(cache *cache.OrderCache, db *sql.DB) http.HandlerFunc {
// 	// Предзагружаем шаблон при старте
// 	tmpl, err := loadTemplate()
// 	if err != nil {
// 		log.Printf("Warning: could not preload template: %v", err)
// 	}

// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Извлекаем order_uid из URL
// 		parts := strings.Split(r.URL.Path, "/")
// 		if len(parts) < 3 {
// 			http.Error(w, "order_uid required", http.StatusBadRequest)
// 			return
// 		}
// 		orderUID := parts[2]

// 		// Валидация orderUID
// 		if !isValidOrderUID(orderUID) {
// 			http.Error(w, "Invalid order_uid format", http.StatusBadRequest)
// 			return
// 		}

// 		// Пробуем достать заказ из кеша
// 		var order domain.Order
// 		var found bool
// 		if cachedOrder, ok := cache.Get(orderUID); ok {
// 			order = cachedOrder
// 			found = true
// 		} else {
// 			// Если в кеше нет — достаем из базы
// 			var err error
// 			order, err = postgres.GetOrderFromDB(db, orderUID)
// 			if err != nil {
// 				if err == sql.ErrNoRows {
// 					log.Printf("Order %s not found in DB", orderUID)
// 					// Используем шаблон для отображения ошибки
// 					renderOrderNotFound(w, orderUID)
// 				} else {
// 					log.Printf("Error loading order %s from DB: %v", orderUID, err)
// 					http.Error(w, "Internal server error", http.StatusInternalServerError)
// 				}
// 				return
// 			}
// 			// Сохраняем в кеш для ускорения следующих запросов
// 			cache.Set(order)
// 			found = true
// 		}

// 		// Если шаблон не был загружен при старте, пробуем загрузить сейчас
// 		if tmpl == nil {
// 			var err error
// 			tmpl, err = loadTemplate()
// 			if err != nil {
// 				log.Printf("Error loading template: %v", err)
// 				http.Error(w, "Internal server error", http.StatusInternalServerError)
// 				return
// 			}
// 		}

// 		// Создаем данные для шаблона
// 		templateData := struct {
// 			Order domain.Order
// 			Found bool
// 		}{
// 			Order: order,
// 			Found: found,
// 		}

// 		w.Header().Set("Content-Type", "text/html; charset=utf-8")
// 		if err := tmpl.Execute(w, templateData); err != nil {
// 			log.Printf("Error executing template: %v", err)
// 			http.Error(w, "Internal server error", http.StatusInternalServerError)
// 		}
// 	}
// }

// renderOrderNotFound рендерит страницу с ошибкой "заказ не найден"
func renderOrderNotFound(w http.ResponseWriter, orderUID string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)

	html := `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Order Not Found</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; }
        .error { color: #d32f2f; margin: 20px 0; }
        .back-link { color: #1976d2; text-decoration: none; }
        .back-link:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>Order Not Found</h1>
    <div class="error">Заказ с ID "` + orderUID + `" не найден</div>
    <a href="/" class="back-link">← Вернуться к поиску</a>
</body>
</html>`

	w.Write([]byte(html))
}
