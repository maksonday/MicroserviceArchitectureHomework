package db

import (
	"fmt"

	"go.uber.org/zap"
)

func ProcessPayment(paymentId, orderId int64) error {
	var (
		accountId int64
		balance   float64
		amount    float64
	)

	if err := GetConn().QueryRow(
		`select a.id, a.balance, p.amount from payments p join orders o on p.order_id = o.id join accounts a on o.user_id = a.user_id 
		where p.id = $1 and o.id = $2 and p.status = 'pending' and o.status = 'pending'`, paymentId, orderId).
		Scan(&accountId, &balance, &amount); err != nil {
		return fmt.Errorf("get account balance: %w", err)
	}

	if int64(balance*100) < int64(amount*100) {
		return fmt.Errorf("insufficient funds")
	}

	if _, err := GetConn().Exec(`update accounts set balance = balance - $1 where id = $2`,
		amount, accountId); err != nil {
		return fmt.Errorf("update account: %w", err)
	}

	if _, err := GetConn().Exec(`update payments set status = 'ok', mtime = NOW() where id = $1`, paymentId); err != nil {
		// Rollback
		GetConn().Exec(`update accounts set balance = balance + $1 where id = $2`, amount, accountId)
		return fmt.Errorf("update payment status: %w", err)
	}

	return nil
}

func RejectPayment(paymentId int64, reason string) {
	if _, err := GetConn().Exec(`update payments set status = 'failed', error = $1, mtime = NOW() where id = $2 and status = 'pending'`, reason, paymentId); err != nil {
		zap.L().Error("failed to reject payment", zap.Error(err))
	}
}
