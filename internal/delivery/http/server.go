package httpdelivery

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"WBtech_l0/internal/config"
	"WBtech_l0/internal/domain"
	"WBtech_l0/internal/repository/cache"
	"WBtech_l0/internal/telemetry"
)

// DBPinger используется для health check, позволяет замокировать БД
type DBPinger interface {
	PingContext(ctx context.Context) error
}
type Server struct {
	cfg     *config.Config
	usecase domain.OrderUsecase
	db      DBPinger
	cache   *cache.OrderCache
	router  *http.ServeMux
	server  *http.Server
}

// NewServer создает новый экземпляр сервера
func NewServer(cfg *config.Config, usecase domain.OrderUsecase, db DBPinger, cache *cache.OrderCache) *Server {
	s := &Server{
		cfg:     cfg,
		usecase: usecase,
		db:      db,
		cache:   cache,
		router:  http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes настраивает маршруты
func (s *Server) setupRoutes() {
	// Создаём middleware для измерения времени запросов (метрики)
	metricsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			duration := time.Since(start).Seconds()
			telemetry.HTTPRequestDuration.WithLabelValues(r.URL.Path, r.Method).Observe(duration)
		})
	}

	// HTML интерфейс (существующий)
	s.router.Handle("/order/", otelhttp.NewHandler(
		metricsMiddleware(MakeOrderHandler(s.usecase)),
		"http-request",
	))

	// JSON API (новые маршруты)
	s.router.Handle("/api/order/", otelhttp.NewHandler(
		metricsMiddleware(MakeJSONOrderHandler(s.usecase)),
		"http-request",
	))
	s.router.Handle("/api/health", otelhttp.NewHandler(
		metricsMiddleware(MakeJSONHealthHandler(s.cache, s.db)),
		"http-request",
	))
	//  Статические файлы и главная страница
	s.router.HandleFunc("/", s.staticFileHandler)
}

// staticFileHandler обрабатывает статические файлы
func (s *Server) staticFileHandler(w http.ResponseWriter, r *http.Request) {
	// Если запрос к API order, пропускаем его к order handler
	if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
		http.NotFound(w, r)
		return
	}

	// Если запрос к HTML order, пропускаем его к order handler
	if len(r.URL.Path) >= 7 && r.URL.Path[:7] == "/order/" {
		MakeOrderHandler(s.usecase)(w, r)
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

	// Создаем http.Server с таймаутами
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting HTTP server at %s\n", addr)
	log.Printf("Web interface available at http://%s\n", addr)
	log.Printf("HTML order view: http://%s/order/{order_uid}\n", addr)
	log.Printf("JSON API: http://%s/api/order/{order_uid}\n", addr)
	log.Printf("Health check: http://%s/api/health\n", addr)
	log.Printf("Serving static files from: web/\n")
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

// Shutdown с использованием http.Server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")
	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			if err != http.ErrServerClosed {
				return fmt.Errorf("server shutdown error: %w", err)
			}
		}
	}
	return nil
}
