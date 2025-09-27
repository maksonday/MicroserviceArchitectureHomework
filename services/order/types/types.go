package types

type HTTPError struct {
	Error string `json:"error"`
}

type Order struct {
	ID      int64  `json:"id,omitempty"`
	Items   []Item `json:"items"`
	Status  string `json:"status,omitempty"`
	Address string `json:"address"`
}

type Item struct {
	Id       int64 `json:"id"`
	Quantity int64 `json:"quantity"`
	StockID  int64
	OrderID  int64
}
