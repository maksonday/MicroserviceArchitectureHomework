package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"order/types"

	"go.uber.org/zap"
)

var ErrEmptyOrder = errors.New("empty order")

func GetUserByOrderID(orderID int64) (int64, error) {
	var userID int64
	if err := GetConn().QueryRow(`select user_id from orders where id = $1`, orderID).Scan(&userID); err != nil {
		return 0, err
	}

	return userID, nil
}

func GetOrders(userID int64) ([]types.Order, error) {
	rows, err := GetConn().Query(`select id, items, status from orders where user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]types.Order, 0)
	for rows.Next() {
		var id int64
		var items, status string
		if err := rows.Scan(&id, &items, &status); err != nil {
			return nil, err
		}

		order := types.Order{
			ID:     id,
			Status: status,
		}

		if err := json.Unmarshal([]byte(items), &order.Items); err != nil {
			zap.L().Error("failed to unpack items", zap.Error(err), zap.String("data", items))
			continue
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func CreateOrder(userID, mask int64, order *types.Order) (int64, error) {
	if len(order.Items) == 0 {
		return 0, ErrEmptyOrder
	}

	for _, item := range order.Items {
		if err := validateItem(&item); err != nil {
			return 0, fmt.Errorf("validate item: %w", err)
		}
	}
	packedItems, err := json.Marshal(order.Items)
	if err != nil {
		return 0, fmt.Errorf("failed to pack items: %w", err)
	}

	var orderID int64
	if err := GetConn().QueryRow(
		`insert into orders(user_id, items, hour_mask) values($1, $2, $3) returning id`, userID, string(packedItems), mask).
		Scan(&orderID); err != nil {
		return 0, fmt.Errorf("failed to create order: %w", err)
	}

	zap.L().Sugar().Infof("order %d created", orderID)

	return orderID, nil
}

func validateItem(item *types.Item) error {
	if item.Quantity < 1 {
		return fmt.Errorf("item %d quantity is non-positive", item.Id)
	}

	var exists bool
	if err := GetConn().QueryRow(`select exists(select 1 from items where id = $1)`, item.Id).Scan(&exists); err != nil {
		return fmt.Errorf("item %d not exists", item.Id)
	}

	return nil
}

func ApproveOrder(orderID int64) error {
	if _, err := GetConn().Exec(`update orders set status = 'approved' where id = $1`, orderID); err != nil {
		return fmt.Errorf("approve order: %w", err)
	}

	zap.L().Sugar().Infof("order %d approved", orderID)

	return nil
}

func RejectOrder(orderID int64) {
	if _, err := GetConn().Exec(`update orders set status = 'canceled' where id = $1`, orderID); err != nil {
		zap.L().Sugar().Errorf("failed to reject order %d: %w", orderID, err)
		return
	}
	zap.L().Sugar().Infof("order %d rejected", orderID)
}
