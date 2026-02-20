# WBtech_l0. Order Service

Микросервис для обработки и отображения данных о заказах, написанный на Go. Сервис получает данные из Kafka, сохраняет в PostgreSQL, кэширует в памяти и предоставляет как HTML интерфейс, так и JSON API.

### Функциональность
- **Получение сообщений** о заказах из Kafka с автоматическим подтверждением
- **Сохранение данных** в PostgreSQL с использованием транзакций
- **In-memory кэширование** с TTL (время жизни) для быстрого доступа
- **Восстановление кеша** из БД при перезапуске сервиса
- **HTML интерфейс** для визуального просмотра заказов
- **JSON API** с форматированным выводом для интеграций
- **Graceful shutdown** для корректного завершения работы
- **Валидация данных** на уровне доменной модели
- **Health check** эндпоинт для мониторинга


### Технологии:  
- **Go 1.23+** - основной язык программирования
- **PostgreSQL** - реляционная база данных
- **Apache Kafka** - брокер сообщений
- **Docker** - контейнеризация (опционально)

WBtech_l0/  
├── cmd/  
│ ├── api/  
│ │ └── main.go             # Точка входа API сервера  
│ └── seed/  
│ └── main.go               # Скрипт для заполнения тестовыми данными  
├── internal/  
│ ├── domain/  
│ │ └── order.go            # Модели данных и валидация  
│ ├── usecase/  
│ │ └── kafka/  
│ │ └── consumer.go         # Kafka consumer  
│ ├── repository/  
│ │ ├── postgres/  
│ │ │ └── order_repo.go     # Работа с БД  
│ │ └── cache/  
│ │ └── memory_cache.go     # In-memory кэш  
│ ├── delivery/  
│ │ └── http/  
│ │ ├── handler.go          # HTML обработчики  
│ │ ├── json_handler.go     # JSON API обработчики  
│ │ └── server.go           # HTTP сервер  
│ └── config/  
│ └── config.go             # Конфигурация  
├── web/  
│ ├── index.html            # Главная страница  
│ └── order_template.html   # Шаблон для отображения заказа  
├── configs/  
│ └── config.yaml           # Конфигурационный файл  
├── migrations/  
│ └── init.sql              # SQL для создания таблиц  
├── Makefile                # Команды для управления проектом  
├── go.mod  
├── go.sum  
└── README.md  


### Запуск

**Запуск PostgreSQL:**  
 Создание базы данных и пользователя  
`sudo -u postgres psql -f migrations/init.sql`  

Или через make:  
****make migrate****

**Запуск Kafka:**
```
# Запуск ZooKeeper (в отдельном терминале):
cd kafka_2.13-3.6.1
bin/zookeeper-server-start.sh config/zookeeper.properties

# Запуск Kafka (в другом терминале)
cd kafka_2.13-3.6.1
bin/kafka-server-start.sh config/server.properties
```

**Запуск сервиса:**  
`go run cmd/main.go`

 Или через make  
****make run-api****


### Использование:
1. Получение заказа в JSON:  
`curl http://localhost:8080/api/order/<order_uid>`  

2. Через веб-интерфейс:
Откройте http://localhost:8080 и введите один из ID:
```
test-order-001
test-order-002
b563feb7b2b84b6test
test-order-empty-items
test-order-multiple-items
```

### Makefile команды

|Команда|Описание|
|:---------------|:---------|
|make run-api| Запуск API сервера|
|make run-seed|	Заполнение БД тестовыми данными|
|make migrate| Применение миграций|
|make build| Сборка бинарных файлов|
|make clean| Очистка бинарных файлов|
|make tidy| Очистка go.mod|


### Особенности реализации
- Кэширование: In-memory кэш с TTL 1 час и максимальным размером 1000 записей

- Валидация: Проверка email, обязательных полей, форматов данных

- Транзакции: Атомарное сохранение заказа со всеми связанными данными

- Graceful shutdown: Корректное завершение при получении сигналов SIGINT/SIGTERM

- Обработка ошибок: Игнорирование невалидных сообщений Kafka, повторные попытки при сбоях