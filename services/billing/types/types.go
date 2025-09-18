package types

type Account struct {
	Id      int64  `db:"id"`
	UserId  string `db:"user_id"`
	Balance int64  `db:"balance"`
}

type Payment struct {
	Amount int64 `json:"amount"`
}

type Deposit struct {
	Amount int64 `json:"amount"`
}

type HTTPError struct {
	Error string `json:"error"`
}

type BalanceResponse struct {
	Balance float64 `json:"balance"`
}
