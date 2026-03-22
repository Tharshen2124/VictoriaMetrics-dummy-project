package models

import "time"

// OrderStatus enumerates the allowed lifecycle states of an order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type OrderItem struct {
	ID			string  `json:"id"`
	OrderID		string  `json:"order_id"`
	ProductID 	string  `json:"product_id"`
	Quantity  	int     `json:"quantity"`
	UnitPrice 	float64 `json:"unit_price"`
}

type Order struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
	Status      OrderStatus `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
}