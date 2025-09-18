package db

import (
	"database/sql"
	"errors"
	"fmt"
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

func ProcessPayment(userId int64, amount int64) error {
	var balance int64
	if err := GetConn().QueryRow(`select balance from accounts where id = $1`, userId).Scan(&balance); err != nil {
		return fmt.Errorf("get account balance: %w", err)
	}

	if balance < amount {
		return fmt.Errorf("insufficient funds")
	}

	if _, err := GetConn().Exec(`update accounts set balance = balance - $1 where id = $2`,
		amount, userId); err != nil {
		return fmt.Errorf("update account: %w", err)
	}

	return nil
}

func AddMoney(userId int64, amount int64) error {
	if _, err := GetConn().Exec(`update accounts set balance = balance + $1 where user_id = $2`, amount, userId); err != nil {
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
