.PHONY: run-api run-seed migrate build

# Переменные для часто используемых значений
BIN_DIR=bin

run-api:
	go run cmd/api/main.go

run-seed:
	go run cmd/seed/main.go

migrate:
	psql -U tmp -d orders_db -f migrations/init.sql

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