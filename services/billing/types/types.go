package types

import "time"

type Account struct {
	Id      int64  `db:"id"`
	UserId  string `db:"user_id"`
	Balance int64  `db:"balance"`
}

type Payment struct {
	ID      int64     `json:"id,omitempty"`
	OrderID int64     `json:"order_id"`
	Amount  float64   `json:"amount"`
	Status  string    `json:"status,omitempty"`
	Action  string    `json:"action,omitempty"`
	CTime   time.Time `json:"ctime"`
	MTime   time.Time `json:"mtime"`
	Error   string    `json:"error,omitempty"`
}

type PaymentsListRequest struct {
	OrderID int64 `json:"order_id"`
}

type Deposit struct {
	Amount float64 `json:"amount"`
}

type HTTPError struct {
	Error string `json:"error"`
}

type BalanceResponse struct {
	Balance float64 `json:"balance"`
}
