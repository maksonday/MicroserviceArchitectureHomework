package db

import (
	"billing/types"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/sethvargo/go-retry"
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

	backoff := retry.WithMaxRetries(retryCount, retry.NewConstant(retryDelay))
	if err := retry.Do(context.Background(), backoff, func(_ context.Context) error {
		var (
			accountID int64
			balance   float64
			amount    float64
			mtime     time.Time
		)

		actionName := "deposit"
		if action == Pay {
			actionName = "pay"
		}

		if err := GetConn().QueryRow(
			`select a.id, a.balance, p.amount, a.mtime from payments p join orders o on p.order_id = o.id join accounts a on o.user_id = a.user_id 
		where p.id = $1 and p.status = 'pending' and p.action = $2`, paymentID, actionName).
			Scan(&accountID, &balance, &amount, &mtime); err != nil {
			return fmt.Errorf("get account balance: %w", err)
		}

		if action == Pay && int64(math.Floor(balance*100)) < int64(math.Ceil(amount*100)) {
			return ErrInsufficientFunds
		}

		if err := processPayment(accountID, amount, action, mtime); err != nil {
			actionName := "pay"
			if action == Deposit {
				actionName = "deposit"
			}

			if errors.Is(err, sql.ErrNoRows) {
				zap.L().Warn("optimistic lock conflict, retrying",
					zap.Int64("payment_id", paymentID),
					zap.Int64("amount", int64(amount)),
				)

				return retry.RetryableError(fmt.Errorf("process payment type "+actionName+": %w", err))
			}

			return fmt.Errorf("process payment type "+actionName+": %w", err)
		}

		return nil
	}); err != nil {
		return err
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

func processPayment(accountID int64, amount float64, action int8, mtime time.Time) error {
	actionType := "-"
	if action == Deposit {
		actionType = "+"
	}

	tx, err := GetConn().BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin tx for payment: %w", err)
	}
	defer tx.Rollback()

	var id int64
	if err := GetConn().QueryRow(`update accounts set balance = balance `+actionType+` $1 where id = $2 and mtime = $3 returning id`,
		amount, accountID, mtime).Scan(&id); err != nil {
		return fmt.Errorf("update account balance: %w", err)
	}

	// если обновилось — коммитим
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
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

func GetPaymentsByOrderID(orderID int64) ([]types.Payment, error) {
	rows, err := GetConn().Query(`select id, action, amount, status, ctime, mtime, error from payments where order_id = $1`, orderID)
	if err != nil {
		return nil, fmt.Errorf("get payments list: %w", err)
	}
	defer rows.Close()

	payments := make([]types.Payment, 0)
	for rows.Next() {
		var (
			id           int64
			action       string
			amount       float64
			status       string
			ctime, mtime time.Time
			Error        string
		)

		if err := rows.Scan(&id, &action, &amount, &status, &ctime, &mtime, &Error); err != nil {
			return nil, fmt.Errorf("scan payments: %w", err)
		}

		payments = append(payments, types.Payment{
			ID:     id,
			Action: action,
			Amount: amount,
			Status: status,
			CTime:  ctime,
			MTime:  mtime,
			Error:  Error,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read payments: %w", err)
	}

	return payments, nil
}
