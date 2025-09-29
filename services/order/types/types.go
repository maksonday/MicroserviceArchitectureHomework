package types

import "time"

type HTTPError struct {
	Error string `json:"error"`
}

type Order struct {
	ID        int64     `json:"id,omitempty"`
	Items     []Item    `json:"items"`
	Status    string    `json:"status,omitempty"`
	Address   string    `json:"address,omitempty"`
	StartTime string    `json:"start_time"`
	EndTime   string    `json:"end_time"`
	Error     string    `json:"error,omitempty"`
	CTime     time.Time `json:"ctime"`
	MTime     time.Time `json:"mtime"`
}

type CreateOrderResponse struct {
	ID int64 `json:"id"`
}

type Item struct {
	Id       int64 `json:"id"`
	Quantity int64 `json:"quantity"`
	StockID  int64 `json:"stock_id,omitempty"`
	OrderID  int64 `json:"order_id,omitempty"`
}
