package httpdelivery

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"WBtech_l0/internal/config"
	"WBtech_l0/internal/repository/cache"
)

type Server struct {
	cfg    *config.Config
	cache  *cache.OrderCache
	db     *sql.DB
	router *http.ServeMux
}

// NewServer создает новый экземпляр сервера
func NewServer(cfg *config.Config, cache *cache.OrderCache, db *sql.DB) *Server {
	s := &Server{
		cfg:    cfg,
		cache:  cache,
		db:     db,
		router: http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes настраивает маршруты
func (s *Server) setupRoutes() {
	// Регистрируем HTTP handler для получения заказа по ID (HTML рендеринг)
	s.router.HandleFunc("/order/", MakeOrderHandler(s.cache, s.db))

	// Обслуживание статических файлов
	s.router.HandleFunc("/", s.staticFileHandler)
}

// staticFileHandler обрабатывает статические файлы
func (s *Server) staticFileHandler(w http.ResponseWriter, r *http.Request) {
	// Если запрос к API order, пропускаем его к order handler
	if len(r.URL.Path) >= 7 && r.URL.Path[:7] == "/order/" {
		MakeOrderHandler(s.cache, s.db)(w, r)
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
}

// Run запускает HTTP сервер
func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%s", s.cfg.HTTPServer.Host, s.cfg.HTTPServer.Port)
	log.Printf("Starting HTTP server at %s\n", addr)
	log.Printf("Web interface available at http://%s\n", addr)
	log.Printf("Serving static files from: web/\n")

	if err := http.ListenAndServe(addr, s.router); err != nil {
		return fmt.Errorf("server failed: %v", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")
	// Здесь можно добавить логику graceful shutdown
	return nil
}
