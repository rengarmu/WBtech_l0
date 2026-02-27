# Makefile для проекта WBtech_l0
# Основные команды:
#   make help          - показать справку
#   make run-api       - запустить API сервер
#   make run-producer  - запустить Kafka продюсер (отправка тестовых сообщений)
#   make run-seed      - запустить наполнение БД тестовыми данными
#   make build         - собрать все бинарники в папку bin/
#   make migrate-up    - применить миграции БД
#   make migrate-down  - откатить последнюю миграцию
#   make test          - запустить юнит-тесты
#   make test-integration - запустить интеграционные тесты (требуют PostgreSQL)
#   make lint          - запустить линтер (golangci-lint)
#   make lint-fix      - запустить линтер с автоматическим исправлением
#   make clean         - удалить собранные бинарники
#   make tidy          - привести go.mod в порядок

.PHONY: help
help:
	@echo "Доступные команды:"
	@echo "  make run-api              - запустить API сервер"
	@echo "  make run-producer          - запустить продюсер (отправка сообщений в Kafka)"
	@echo "  make run-seed              - запустить seed (наполнение БД тестовыми данными)"
	@echo "  make build                 - собрать все бинарники"
	@echo "  make migrate-up            - применить миграции вверх"
	@echo "  make migrate-down          - откатить последнюю миграцию"
	@echo "  make migrate-force V=<n>   - принудительно установить версию миграции"
	@echo "  make migrate-create NAME=<name> - создать новую миграцию"
	@echo "  make test                  - запустить все тесты"
	@echo "  make test-integration      - запустить интеграционные тесты (с тегом integration)"
	@echo "  make lint                  - запустить линтер (golangci-lint)"
	@echo "  make lint-fix              - запустить линтер с автоисправлением"
	@echo "  make clean                 - удалить бинарники"
	@echo "  make tidy                  - go mod tidy"
	
# Параметры по умолчанию (можно переопределить через окружение)
DB_DSN ?= postgres://tmp:test90123@localhost:5432/orders_db?sslmode=disable
MIGRATIONS_PATH ?= ./migrations
BIN_DIR ?= ./bin

# Пути к бинарникам
BINARY_API = $(BIN_DIR)/api
BINARY_PRODUCER = $(BIN_DIR)/producer
BINARY_SEED = $(BIN_DIR)/seed

# Команда для миграций (используем go run, т.к. migrate уже есть в зависимостях)
MIGRATE_CMD = go run -tags migrate github.com/golang-migrate/migrate/v4/cmd/migrate

# ------------------------------------------------------------
# Запуск приложений
# ------------------------------------------------------------
.PHONY: run-api
run-api:
	go run cmd/api/main.go

.PHONY: run-producer
run-producer:
	go run cmd/producer/main.go

.PHONY: run-seed
run-seed:
	go run cmd/seed/main.go

# ------------------------------------------------------------
# Сборка
# ------------------------------------------------------------
.PHONY: build
build: $(BINARY_API) $(BINARY_PRODUCER) $(BINARY_SEED)

$(BINARY_API): cmd/api/main.go
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY_API) cmd/api/main.go

$(BINARY_PRODUCER): cmd/producer/main.go
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY_PRODUCER) cmd/producer/main.go

$(BINARY_SEED): cmd/seed/main.go
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY_SEED) cmd/seed/main.go

# ------------------------------------------------------------
# Миграции
# ------------------------------------------------------------
.PHONY: migrate-up
migrate-up:
	$(MIGRATE_CMD) -database "$(DB_DSN)" -path $(MIGRATIONS_PATH) up

.PHONY: migrate-down
migrate-down:
	$(MIGRATE_CMD) -database "$(DB_DSN)" -path $(MIGRATIONS_PATH) down 1

.PHONY: migrate-force
migrate-force:
	@if [ -z "$(V)" ]; then echo "Ошибка: укажите версию: make migrate-force V=123"; exit 1; fi
	$(MIGRATE_CMD) -database "$(DB_DSN)" -path $(MIGRATIONS_PATH) force $(V)

.PHONY: migrate-create
migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Ошибка: укажите имя миграции: make migrate-create NAME=create_users"; exit 1; fi
	$(MIGRATE_CMD) create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

# ------------------------------------------------------------
# Тестирование
# ------------------------------------------------------------
.PHONY: test
test:
	go test -v -race -cover ./...

.PHONY: test-integration
test-integration:
	go test -v -race -cover -tags=integration ./internal/repository/postgres/...

# ------------------------------------------------------------
# Линтинг
# ------------------------------------------------------------
.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix ./...

# ------------------------------------------------------------
# Утилиты
# ------------------------------------------------------------
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: db-shell
db-shell:
	psql "$(DB_DSN)"

.PHONY: run-all
run-all: migrate-up run-api
	@echo "Сервис запущен. Для остановки нажмите Ctrl+C"