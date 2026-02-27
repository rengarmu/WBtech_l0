
# WBtech L0 — Order Service

Микросервис для приёма, хранения и отображения информации о заказах.  
Сервис получает данные из Kafka, сохраняет в PostgreSQL, кэширует в памяти и предоставляет как HTML-интерфейс, так и JSON API.

## Возможности

- **Получение сообщений** из Kafka (топик `orders`) с автоматическим подтверждением
- **Валидация** входящих данных на уровне доменной модели
- **Dead Letter Queue (DLQ)** — сообщения с ошибками отправляются в отдельный топик
- **Сохранение в PostgreSQL** с использованием транзакций (основной заказ, доставка, оплата, товары)
- **In‑memory кэш** с TTL и ограничением размера, автоматическое восстановление из БД при старте
- **HTML интерфейс** для визуального просмотра заказа по UID
- **JSON API** для интеграции с другими сервисами
- **Метрики Prometheus** (количество обработанных заказов, длительность запросов)
- **Трассировка Jaeger** (OpenTelemetry)
- **Graceful shutdown** — корректное завершение работы
- **Инструменты разработки**: миграции БД, продюсер для отправки тестовых сообщений, скрипт наполнения базы

## Технологии

- **Go** 1.24+
- **PostgreSQL** — хранение данных
- **Apache Kafka** — брокер сообщений
- **golang-migrate** — миграции БД
- **Prometheus** — метрики
- **OTLP (OpenTelemetry Protocol)** — трассировка (OpenTelemetry)
- **Viper** — конфигурация
- **Testify** (опционально) — для тестов

## Структура проекта

```
WBtech_l0/
├── cmd/                             # Точки входа
│   ├── api/                         # API сервер
│   ├── producer/                    # Kafka продюсер
│   └── seed/                        # Наполнение БД тестовыми данными
├── internal/                        # Внутренние пакеты
│   ├── config/                      # Конфигурация
│   ├── delivery/                    # HTTP обработчики
│   │   └── http/
│   │       ├── handler.go           # HTML обработчики
│   │       ├── json_handler.go      # JSON API
│   │       └── server.go            # HTTP сервер
│   ├── domain/                      # Модели и интерфейсы
│   │   ├── interfaces.go
│   │   └── order.go
│   ├── repository/                  # Репозитории
│   │   ├── cache/                   # In‑memory кэш
│   │   └── postgres/                # PostgreSQL
│   ├── telemetry/                   # Метрики и трассировка
│   └── usecase/                     # Бизнес-логика
│       ├── kafka/
│       │   └── consumer.go          # Kafka consumer
│       └── order_usecase.go
├── migrations/                      # SQL миграции
├── web/                             # Статические файлы и шаблоны
│   ├── index.html
│   └── order_template.html
├── configs/                         # Конфигурационные файлы
│   └── config.yaml
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

## Быстрый старт

### Требования

- Go 1.24 или новее
- PostgreSQL (локально или в Docker)
- Kafka (локально или в Docker)

### Настройка

1. Установите зависимости:
   ```bash
   go mod tidy
   ```

2. Создайте базу данных и пользователя в PostgreSQL:
   ```sql
   CREATE DATABASE orders_db;
   CREATE USER tmp WITH PASSWORD 'test90123';
   GRANT ALL PRIVILEGES ON DATABASE orders_db TO tmp;
   ```
   (пароль и пользователь должны совпадать с `configs/config.yaml`)

3. Примените миграции:
   ```bash
   make migrate-up
   ```

4. Запустите Kafka (например, через Docker):
   ```bash
    # Запуск ZooKeeper (в отдельном терминале):
    cd kafka_2.13-3.6.1
    bin/zookeeper-server-start.sh config/zookeeper.properties

    # Запуск Kafka (в другом терминале)
    cd kafka_2.13-3.6.1
    bin/kafka-server-start.sh config/server.properties
   ```

6. (Опционально) Наполните базу тестовыми данными:
   ```bash
   make run-seed
   ```

7. Запустите сервис:
   ```bash
   make run-api
   ```

8. В другом терминале запустите продюсер для отправки тестовых заказов в Kafka:
   ```bash
   make run-producer
   ```

### Проверка работы

- HTML интерфейс: http://localhost:8080
- JSON API: http://localhost:8080/api/order/<order_uid>
- Health check: http://localhost:8080/api/health
- Метрики Prometheus: http://localhost:2112/metrics
- Трассировка OTLP HTTP (если запущен): http://localhost:16686

## Конфигурация

Основные настройки задаются в файле `configs/config.yaml` или через переменные окружения.

```yaml
postgresql:
  host: "localhost"
  port: "5432"
  user: "tmp"
  password: "test90123"
  database: "orders_db"

http_server:
  host: ""
  port: "8080"

kafka:
  brokers: "localhost:9092"
  topic: "orders"
  group_id: "order-service-group"
  dlq_topic: "orders-dlq"

cache:
  default_ttl: 1h
  max_size: 1000

telemetry:
  jaeger_url: "http://localhost:14268/api/traces"
  metrics_port: "2112"
```

Все параметры можно переопределить через переменные окружения с префиксом (например, `POSTGRESQL_HOST`).

## API Endpoints

| Метод | Путь                  | Описание                          |
|-------|-----------------------|-----------------------------------|
| GET   | `/order/{order_uid}`  | HTML страница с деталями заказа   |
| GET   | `/api/order/{order_uid}` | JSON данные заказа              |
| GET   | `/api/health`         | Статус сервиса (БД, кэш)          |
| GET   | `/metrics`            | Метрики Prometheus                |


## Команды Makefile

| Команда                 | Описание                                    |
|-------------------------|---------------------------------------------|
| `make help`             | Показать справку                            |
| `make run-api`          | Запустить API сервер                         |
| `make run-producer`     | Запустить Kafka продюсер (отправка тестовых сообщений) |
| `make run-seed`         | Наполнить БД тестовыми данными               |
| `make build`            | Собрать все бинарники в папку `bin/`         |
| `make migrate-up`       | Применить миграции                           |
| `make migrate-down`     | Откатить последнюю миграцию                  |
| `make test`             | Запустить юнит‑тесты                         |
| `make test-integration` | Запустить интеграционные тесты (требуют PostgreSQL) |
| `make lint`             | Запустить линтер (`golangci-lint`)           |
| `make lint-fix`         | Запустить линтер с автоматическим исправлением |
| `make tidy`             | `go mod tidy`                                |
| `make clean`            | Удалить собранные бинарники                  |

## Тестирование

### Юнит-тесты
```bash
make test
```

### Интеграционные тесты (PostgreSQL)
```bash
make test-integration
```

Интеграционные тесты используют отдельную базу данных `orders_db_test`. Убедитесь, что она создана и доступна.

## Линтинг

Проект использует `golangci-lint` с набором популярных линтеров. Конфигурация в файле `.golangci.yml`.

```bash
make lint      # только проверка
make lint-fix  # проверка + автоисправление
```

