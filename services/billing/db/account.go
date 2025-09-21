package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
)

var (
	ErrNoUser         = errors.New("user not found")
	ErrNotEnoughFunds = errors.New("not enough funds")
)

func CreateAccount(userId int64) (int64, error) {
	res := GetConn().QueryRow(`insert into accounts(user_id) values($1) returning id`, userId)

	var id int64
	if err := res.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("create account: no rows returned")
		}
		return 0, fmt.Errorf("create account: %w", err)
	}

	return id, nil
}

func AddMoney(userId int64, amount float64) error {
	rounded := math.Floor(amount*100) / 100
	if _, err := GetConn().Exec(`update accounts set balance = balance + $1 where user_id = $2`, rounded, userId); err != nil {
		return fmt.Errorf("add money: %w", err)
	}

	return nil
}

func GetBalance(userId int64) (float64, error) {
	var balance float64
	if err := GetConn().QueryRow(`select balance from accounts where user_id = $1`, userId).Scan(&balance); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNoUser
		}
		return 0, fmt.Errorf("get account balance: %w", err)
	}

	return balance, nil
}
