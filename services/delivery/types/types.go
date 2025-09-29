package types

import "time"

type Courier struct {
	Name string `json:"name"`
}

type CourierReservation struct {
	ID        int64     `json:"id"`
	OrderID   int64     `json:"order_id"`
	CourID    int64     `json:"courier_id"`
	Action    string    `json:"action"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Error     string    `json:"error,omitempty"`
	CTime     time.Time `json:"ctime"`
	MTime     time.Time `json:"mtime"`
}

type CourReserveListRequest struct {
	OrderID int64 `json:"order_id"`
}

type Order struct {
	OrderID int64 `json:"order_id"`
}

type HTTPError struct {
	Error string `json:"error"`
}
