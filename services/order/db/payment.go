package db

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

func CreatePayment(orderID int64, stockChangeIDs []int64) (int64, error) {
	totalPrice, err := calculateOrderTotalPrice(stockChangeIDs)
	if err != nil {
		return 0, fmt.Errorf("calculate order total price: %w", err)
	}

	var paymentID int64
	if err := GetConn().QueryRow(
		`insert into payments(order_id, action, amount) values($1, 'pay', $2) returning id`, orderID, totalPrice).Scan(&paymentID); err != nil {
		return 0, fmt.Errorf("create payment: %w", err)
	}

	zap.L().Sugar().Infof("payment %d created", paymentID)

	return paymentID, nil
}

func calculateOrderTotalPrice(stockChangeIDs []int64) (float64, error) {
	changes := changesToStr(stockChangeIDs)
	rows, err := GetConn().Query(
		fmt.Sprintf(`select sc.quantity, i.price from stock_changes sc 
		join stock s on s.id = sc.stock_id 
		join items i on i.id = s.item_id 
		where sc.id in (%s)`, strings.Join(changes, ",")))
	if err != nil {
		return 0, fmt.Errorf("failed to get stock_changes: %w", err)
	}
	defer rows.Close()

	var totalPrice float64
	for rows.Next() {
		var (
			quantity int64
			price    float64
		)
		if err := rows.Scan(&quantity, &price); err != nil {
			return 0, fmt.Errorf("scan stock_changes: %w", err)
		}

		totalPrice += float64(quantity) * price
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("stock changes rows err: %w", err)
	}

	return math.Ceil(totalPrice*100) / 100, nil
}

func changesToStr(changes []int64) []string {
	changesStr := make([]string, 0, len(changes))
	for _, id := range changes {
		changesStr = append(changesStr, strconv.FormatInt(id, 10))
	}

	return changesStr
}

func buildRevertPayment(paymentID int64) (int64, float64, error) {
	var (
		orderID int64
		amount  float64
	)
	if err := GetConn().QueryRow(`select order_id, amount from payments where id = $1 and action = 'pay'`, paymentID).
		Scan(&orderID, &amount); err != nil {
		return 0, 0, err
	}

	return orderID, amount, nil
}

func RevertPayment(paymentID int64) (int64, error) {
	orderID, amount, err := buildRevertPayment(paymentID)
	if err != nil {
		return 0, fmt.Errorf("build revert payment: %w", err)
	}

	var newID int64
	if err := GetConn().QueryRow(
		`insert into payments(order_id, amount, action) values ($1, $2, 'deposit') returning id`, orderID, amount).Scan(&newID); err != nil {
		return 0, err
	}

	return newID, nil
}
