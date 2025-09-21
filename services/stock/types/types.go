package types

type HTTPError struct {
	Error string `json:"error"`
}

type Item struct {
	Id          int64   `db:"id" json:"id,omitempty"`
	Name        string  `db:"name" json:"name"`
	Description string  `db:"description" json:"description"`
	Price       float64 `db:"price" json:"price"`
}

type StockChange struct {
	ItemID   int64 `db:"item_id" json:"item_id"`
	Action   int8  `db:"action" json:"action"`
	Quantity int64 `db:"quantity" json:"quantity"`
	StockId  int64
}

type ItemsResponse struct {
	Items []Item `json:"items"`
}
