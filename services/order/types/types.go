package types

type Order struct {
	ID    int64
	Items []Item `json:"items"`
}

type Item struct {
	Id       int64 `json:"id"`
	Quantity int64 `json:"quantity"`
	StockID  int64
	OrderID  int64
}

type CreateOrderResponse struct {
}
