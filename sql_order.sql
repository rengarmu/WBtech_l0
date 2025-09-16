CREATE DATABASE orders_db;

create user tmp with password 'test90123';

alter database orders_db owner to tmp;

-- Создание таблицы заказов
CREATE TABLE orders (
    order_uid VARCHAR(255) PRIMARY KEY not null,
    track_number VARCHAR(255) not null,
    entry VARCHAR(50) not null,
    locale VARCHAR(10) not null,
    internal_signature TEXT,
    customer_id VARCHAR(255) not null,
    delivery_service VARCHAR(255) not null,
    shardkey VARCHAR(10) not null,
    sm_id INTEGER not null,
    date_created TIMESTAMP WITH TIME ZONE NOT NULL,
    oof_shard VARCHAR(10) not null
);

-- Данные доставки (связь с orders)
CREATE TABLE deliveries (
    id SERIAL PRIMARY KEY,
    order_uid VARCHAR(255) NOT NULL REFERENCES orders (order_uid) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) NOT NULL,
    zip VARCHAR(20) NOT NULL,
    city VARCHAR(100) NOT NULL,
    address TEXT NOT NULL,
    region VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL
);

-- Данные оплаты (связь с orders)
CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    order_uid VARCHAR(255) NOT NULL REFERENCES orders (order_uid) ON DELETE CASCADE,
    transaction VARCHAR(255) NOT NULL,
    request_id VARCHAR(255),
    currency VARCHAR(10) NOT NULL,
    provider VARCHAR(100) NOT NULL,
    amount INTEGER NOT NULL,
    payment_dt BIGINT NOT NULL,
    bank VARCHAR(100) NOT NULL,
    delivery_cost INTEGER NOT NULL,
    goods_total INTEGER NOT NULL,
    custom_fee INTEGER NOT NULL
);

-- Товары (связь с orders)
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    order_uid VARCHAR(255) NOT NULL REFERENCES orders (order_uid) ON DELETE CASCADE,
    chrt_id BIGINT NOT NULL,
    track_number VARCHAR(255) NOT NULL,
    price INTEGER NOT NULL,
    rid VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    sale INTEGER NOT NULL,
    size VARCHAR(50) NOT NULL,
    total_price INTEGER NOT NULL,
    nm_id BIGINT NOT NULL,
    brand VARCHAR(255) NOT NULL,
    status INTEGER NOT NULL
);

SELECT current_database();

SELECT current_user;

GRANT CONNECT ON DATABASE orders_db TO tmp;

SET ROLE tmp;