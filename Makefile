.PHONY: run-api run-seed migrate build

# Переменные для часто используемых значений
BIN_DIR=bin
MIGRATE = migrate -path ./migrations -database "postgres://tmp:test90123@localhost:5432/orders_db?sslmode=disable"

run-api:
	go run cmd/api/main.go

run-seed:
	go run cmd/seed/main.go

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down

migrate-force:
	$(MIGRATE) force $(V)

migrate-version:
	$(MIGRATE) version

run-producer:
	go run cmd/producer/main.go

run-producer-valid:
	go run cmd/producer/main.go -type=valid -count=5

run-producer-invalid:
	go run cmd/producer/main.go -type=invalid -count=3

kafka-create-topics:
	# Проверяем наличие утилиты kafka-topics
	@command -v kafka-topics >/dev/null 2>&1 || { echo "kafka-topics not found. Please ensure Kafka binaries are in PATH."; exit 1; }
	# Создаём топик orders, если не существует
	kafka-topics --create --topic orders \
		--bootstrap-server localhost:9092 \
		--partitions 1 --replication-factor 1 || true
	# Создаём топик orders-dlq, если не существует
	kafka-topics --create --topic orders-dlq \
		--bootstrap-server localhost:9092 \
		--partitions 1 --replication-factor 1 || true

build:
	go build -o bin/api cmd/api/main.go
	go build -o bin/seed cmd/seed/main.go

clean:
	rm -rf $(BIN_DIR)/

tidy:
	go mod tidy


help:
	@echo "Available commands:"
	@echo "  make run-api     - Запуск API сервера"
	@echo "  make run-seed    - Заполнение БД тестовыми данными"
	@echo "  make migrate     - Применение миграций"
	@echo "  make build       - Сборка бинарных файлов"
	@echo "  make clean       - Очистка бинарных файлов"
	@echo "  make tidy        - Очистка go.mod"


.DEFAULT_GOAL := run-api