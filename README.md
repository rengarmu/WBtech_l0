# WBtech_l0. Order Service


Микросервис для обработки и отображения данных о заказах, написанный на Go.

### Функциональность
- Получение сообщений о заказах из Kafka

- Сохранение данных в PostgreSQL

- In-memory кэширование для быстрого доступа

- HTTP API для получения информации о заказах

- Веб-интерфейс для поиска заказов

### Технологии:  

**Go** - основной язык программирования  
**PostgreSQL** - реляционная база данных  
**Apache Kafka** - брокер сообщений

WBtech_l0/  
├── cmd/  
│   └── main.go  
├── /backend   
│   ├── order_handler.go  
│   ├── cache.go  
│   ├── config.go  
│   ├── database.go  
│   ├── model.go 
│   └── kafka.go  
├── web/   
│   └── index.html  
├── sql_orders.sql  
├── go.mod  
└── README.md  


### Запуск

**Запуск PostgreSQL:**  
`sudo service postgresql start`

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


### Использование:
1. Получить информацию о заказе через **HTTP API**:  
`curl http://localhost:8080/order/<order_uid>`
2. Откройте в браузере:  
http://localhost:8080/order/<order_uid>

### Для тестирования:
После запуска скрипта вы можете проверить работу:

**Через веб-интерфейс:**   
Откройте http://localhost:8080 и введите один из ID:
```
test-order-001
test-order-002
b563feb7b2b84b6test
test-order-empty-items
test-order-multiple-items
```

**Через HTTP API:**

bash  
`curl http://localhost:8080/order/test-order-001`  
`curl http://localhost:8080/order/test-order-multiple-items`