CREATE DATABASE orders_db

create user alabeshko with password '613panda'

alter database orders_db owner to alabeshko

create table if NOT exists orders (
    order_uid varchar(255) PRIMARY KEY not null, -- UID заказа
    track_number varchar(255) not null, -- Трек-номер
    entry VARCHAR(255) not null, -- Вход
    locale varchar(10) not null, -- Локаль
    internal_signature varchar(255) not null, -- Внутреняя подпись, комментарий (может быть пустой)
    customer_id varchar(255) not null, -- ID покупателя
    delivery_service varchar(255) not null, -- Служба доставки
    shardkey varchar(10) not null, -- Шард-ключ
    sm_id INTEGER not null, -- SM ID
    date_created TIMESTAMP WITH TIME ZONE NOT NULL, -- Дата создания (с учетом часового пояса)
    oof_shard varchar(10) not null -- OOF шард
);

create table if not exists delivery (
    id SERIAL PRIMARY KEY, -- Суррогатный первичный ключ
    order_uid VARCHAR(255) not NULL REFERENCES orders (order_uid) on delete CASCADE, -- Связь с заказом 
    name VARCHAR(255) NOT NULL, -- ФИО получателя
    phone VARCHAR(50) NOT NULL, -- Телефон
    zip VARCHAR(50) NOT NULL, -- Почтовый индекс
    city VARCHAR(255) NOT NULL, -- Город
    address VARCHAR(255) NOT NULL, -- Адрес
    region VARCHAR(255) NOT NULL, -- Регион
    email VARCHAR(255) NOT NULL -- Email
);

create table if not exists payment (
    id SERIAL PRIMARY KEY, -- Суррогатный первичный ключ    
    order_uid VARCHAR(255) NOT NULL REFERENCES orders (order_uid) on delete CASCADE, -- Связь с заказом
    transaction VARCHAR(255) NOT NULL, -- ID транзакции
    request_id VARCHAR(255), -- ID запроса (может быть пустым)
    currency VARCHAR(10) NOT NULL, -- Валюта
    provider VARCHAR(50) NOT NULL, -- Провайдер платежа
    amount INTEGER NOT NULL, -- Сумма
    payment_dt BIGINT NOT NULL, -- Дата и время платежа (Unix time)
    bank VARCHAR(255) NOT NULL, -- Банк
    delivery_cost INTEGER NOT NULL, -- Стоимость доставки
    goods_total INTEGER NOT NULL, -- Общая стоимость товаров
    custom_fee INTEGER NOT NULL -- Комиссия
);

create table if not exists items (
    id SERIAL PRIMARY KEY, -- Суррогатный первичный ключ
    order_uid VARCHAR(255) NOT NULL REFERENCES orders (order_uid) ON DELETE CASCADE, -- Связь с заказом
    chrt_id INTEGER NOT NULL, -- Числовой ID товара
    track_number VARCHAR(255) NOT NULL, -- Трек-номер для этого товара
    price INTEGER NOT NULL, -- Цена за единицу
    rid VARCHAR(255) NOT NULL, -- RID
    item_name VARCHAR(255) NOT NULL, -- Наименование
    sale INTEGER NOT NULL, -- Скидка
    size VARCHAR(50) NOT NULL, -- Размер
    total_price INTEGER NOT NULL, -- Общая цена (с учётом количества и скидки)
    nm_id INTEGER NOT NULL, -- NM ID
    brand VARCHAR(255) NOT NULL, -- Бренд
    status INTEGER NOT NULL -- Статус
)

SELECT current_database();

SELECT current_user;

GRANT CONNECT ON DATABASE orders_db TO alabeshko;