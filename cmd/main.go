package main

import (
	"WBtech_l0/backend"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

	// Регистрируем HTTP handler для получения заказа по ID (HTML рендеринг)
	http.HandleFunc("/order/", backend.MakeOrderHandler(cache, db))

	// Обслуживание статических файлов из папки web
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Если запрос к API order, пропускаем его к order handler
		if len(r.URL.Path) >= 7 && r.URL.Path[:7] == "/order/" {
			backend.MakeOrderHandler(cache, db)(w, r)
			return
		}

		// Определяем путь к файлу
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Путь к папке web
		webPath := filepath.Join("web", path)

		// Проверяем существование файла
		if _, err := os.Stat(webPath); os.IsNotExist(err) {
			// Если файл не найден, отдаем index.html (для SPA routing)
			http.ServeFile(w, r, filepath.Join("web", "index.html"))
			return
		}

		// Отдаем статический файл
		http.ServeFile(w, r, webPath)
	})

	// Запускаем HTTP сервер
	addr := fmt.Sprintf("%s:%s", cfg.HTTPServer.Host, cfg.HTTPServer.Port)
	log.Printf("Starting HTTP server at %s\n", addr)
	log.Printf("Web interface available at http://%s\n", addr)
	log.Printf("Serving static files from: web/\n")
	log.Fatal(http.ListenAndServe(addr, nil))
}
