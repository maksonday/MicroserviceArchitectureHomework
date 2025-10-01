package types

import "time"

type HTTPError struct {
	Error string `json:"error"`
}

type Item struct {
	Id          int64   `db:"id" json:"id,omitempty"`
	Name        string  `db:"name" json:"name"`
	Description string  `db:"description" json:"description"`
	Price       float64 `db:"price" json:"price"`
	Quantity    int64   `json:"quantity,omitempty"`
}

type StockChange struct {
	ItemID   int64     `db:"item_id" json:"item_id,omitempty"`
	Action   string    `db:"action" json:"action"`
	Quantity int64     `db:"quantity" json:"quantity"`
	StockId  int64     `json:"stock_id,omitempty"`
	OrderID  int64     `json:"order_id,omitempty"`
	ID       int64     `json:"id,omitempty"`
	CTime    time.Time `json:"ctime"`
	MTime    time.Time `json:"mtime"`
	Status   string    `json:"status,omitempty"`
	Error    string    `json:"error,omitempty"`
}

type StockChangesListRequest struct {
	OrderID int64 `json:"order_id"`
}

type ItemsResponse struct {
	Items []Item `json:"items"`
}
