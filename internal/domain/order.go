package domain

import (
	"errors"
	"regexp"
	"time"
)

// Order — основная модель заказа
type Order struct {
	OrderUID          string   `json:"order_uid"`
	TrackNumber       string   `json:"track_number"`
	Entry             string   `json:"entry"`
	Delivery          Delivery `json:"delivery"`
	Payment           Payment  `json:"payment"`
	Items             []Item   `json:"items"`
	Locale            string   `json:"locale"`
	InternalSignature string   `json:"internal_signature"`
	CustomerID        string   `json:"customer_id"`
	DeliveryService   string   `json:"delivery_service"`
	Shardkey          string   `json:"shardkey"`
	SmID              int      `json:"sm_id"`
	DateCreated       string   `json:"date_created"`
	OofShard          string   `json:"oof_shard"`
}

// Delivery — информация о доставке
type Delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

// Payment — информация об оплате
type Payment struct {
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDT    int64  `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

// Item — информация о товаре
type Item struct {
	ChrtID      int    `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	Rid         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NmID        int    `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

// Validate проверяет корректность заказа
func (o *Order) Validate() error {
	if o.OrderUID == "" {
		return errors.New("order_uid is required")
	}
	if len(o.OrderUID) > 100 {
		return errors.New("order_uid too long")
	}

	if o.TrackNumber == "" {
		return errors.New("track_number is required")
	}

	if o.Entry == "" {
		return errors.New("entry is required")
	}

	// Валидация даты
	if o.DateCreated != "" {
		if _, err := time.Parse(time.RFC3339, o.DateCreated); err != nil {
			return errors.New("invalid date_created format, expected RFC3339")
		}
	}

	// Валидация вложенных структур
	if err := o.Delivery.Validate(); err != nil {
		return err
	}

	if err := o.Payment.Validate(); err != nil {
		return err
	}

	if len(o.Items) == 0 {
		return errors.New("at least one item is required")
	}

	for i, item := range o.Items {
		if err := item.Validate(); err != nil {
			return errors.New("item " + string(rune(i)) + ": " + err.Error())
		}
	}

	return nil
}

// Validate проверяет корректность данных доставки
func (d *Delivery) Validate() error {
	if d.Name == "" {
		return errors.New("delivery name is required")
	}
	if d.Phone == "" {
		return errors.New("delivery phone is required")
	}
	if d.City == "" {
		return errors.New("delivery city is required")
	}
	if d.Address == "" {
		return errors.New("delivery address is required")
	}
	if d.Email != "" {
		emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
		if !emailRegex.MatchString(d.Email) {
			return errors.New("invalid email format")
		}
	}
	return nil
}

// Validate проверяет корректность данных оплаты
func (p *Payment) Validate() error {
	if p.Transaction == "" {
		return errors.New("payment transaction is required")
	}
	if p.Currency == "" {
		return errors.New("payment currency is required")
	}
	if p.Amount <= 0 {
		return errors.New("payment amount must be positive")
	}
	if p.PaymentDT <= 0 {
		return errors.New("invalid payment timestamp")
	}
	return nil
}

// Validate проверяет корректность данных товара
func (i *Item) Validate() error {
	if i.ChrtID <= 0 {
		return errors.New("chrt_id must be positive")
	}
	if i.TrackNumber == "" {
		return errors.New("item track_number is required")
	}
	if i.Price <= 0 {
		return errors.New("item price must be positive")
	}
	if i.Name == "" {
		return errors.New("item name is required")
	}
	if i.TotalPrice <= 0 {
		return errors.New("item total_price must be positive")
	}
	if i.NmID <= 0 {
		return errors.New("nm_id must be positive")
	}
	return nil
}
