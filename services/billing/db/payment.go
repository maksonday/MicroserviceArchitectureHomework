package db

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
)

const (
	Deposit = iota
	Pay
)

var (
	ErrUnsupportedPaymentAction = errors.New("unsupported payment action")
	ErrInsufficientFunds        = errors.New("insufficient funds")
)

func ProcessPayment(paymentID int64, action int8) error {
	if action != Deposit && action != Pay {
		return ErrUnsupportedPaymentAction
	}

	var (
		accountID int64
		balance   float64
		amount    float64
	)

	actionName := "deposit"
	if action == Pay {
		actionName = "pay"
	}

	if err := GetConn().QueryRow(
		`select a.id, a.balance, p.amount from payments p join orders o on p.order_id = o.id join accounts a on o.user_id = a.user_id 
		where p.id = $1 and p.status = 'pending' and o.status = 'pending' and p.action = $2`, paymentID, actionName).
		Scan(&accountID, &balance, &amount); err != nil {
		return fmt.Errorf("get account balance: %w", err)
	}

	if action == Pay && int64(balance*100) < int64(amount*100) {
		return ErrInsufficientFunds
	}

	if err := processPayment(accountID, amount, action); err != nil {
		actionName := "pay"
		if action == Deposit {
			actionName = "deposit"
		}
		return fmt.Errorf("process payment type "+actionName+": %w", err)
	}

	zap.L().Info("payment processed", zap.Int64("payment_id", paymentID), zap.Int8("action", action))

	return nil
}

func ApprovePayment(paymentID int64) {
	if _, err := GetConn().Exec(`update payments set status = 'ok', mtime = NOW() where id = $1`, paymentID); err != nil {
		zap.L().Error("failed to approve payment", zap.Error(err))
		return
	}

	zap.L().Info("payment approved", zap.Int64("payment_id", paymentID))
}

func processPayment(accountID int64, amount float64, action int8) error {
	actionType := "-"
	if action == Deposit {
		actionType = "+"
	}

	if _, err := GetConn().Exec(`update accounts set balance = balance `+actionType+` $1 where id = $2`,
		amount, accountID); err != nil {
		return fmt.Errorf("update account balance: %w", err)
	}

	zap.L().Info(
		"account updated",
		zap.Int64("account_id", accountID),
		zap.Int8("action", action),
		zap.Float64("amount", amount),
	)

	return nil
}

func RejectPayment(paymentID int64, reason string) {
	if _, err := GetConn().Exec(
		`update payments set status = 'failed', error = $1, mtime = NOW() where id = $2 and status = 'pending'`,
		reason, paymentID); err != nil {
		zap.L().Error("failed to reject payment", zap.Error(err))
		return
	}

	zap.L().Info("payment rejected", zap.Int64("payment_id", paymentID), zap.String("reason", reason))
}
