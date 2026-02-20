.PHONY: run-api run-seed migrate test build

run-api:
	go run cmd/api/main.go

run-seed:
	go run cmd/seed/main.go

migrate:
	psql -U tmp -d orders_db -f migrations/init.sql

test:
	go test -v ./...

build:
	go build -o bin/api cmd/api/main.go
	go build -o bin/seed cmd/seed/main.go

clean:
	rm -rf bin/

.DEFAULT_GOAL := run-api