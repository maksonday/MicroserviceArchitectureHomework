package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"order/types"

	"go.uber.org/zap"
)

var ErrEmptyOrder = errors.New("empty order")

func CreateOrder(userID int64, order *types.Order) (int64, error) {
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
		`insert into orders(user_id, items) values($1, $2) returning id`, userID, string(packedItems)).Scan(&orderID); err != nil {
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
