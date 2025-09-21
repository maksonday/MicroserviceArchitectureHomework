package db

import (
	"errors"
	"fmt"
	"stock/types"
)

const (
	StockChangeAdd = iota
	StockChangeRemove
	StockChangeReserve
)

var (
	ErrUnsupportedStockChangeAction = errors.New("unsupported stock change action")
)

func ProcessStockChange(stockChange *types.StockChange) error {
	var (
		stockId int64
		err     error
	)

	stockId, err = getStockId(stockChange.ItemID)
	if err != nil {
		return err
	}

	stockChange.StockId = stockId
	if err := insertStockChanges(stockChange); err != nil {
		return err
	}

	switch stockChange.Action {
	case StockChangeAdd:
		return addStockItems(stockChange)
	case StockChangeRemove:
		return removeStockItems(stockChange)
	case StockChangeReserve:
		return reserveStockItems(stockChange)
	default:
		return ErrUnsupportedStockChangeAction
	}
}

func getStockId(itemId int64) (int64, error) {
	var stockId int64
	if err := GetConn().QueryRow(`SELECT id from stock where item_id = $1`, itemId).Scan(&stockId); err != nil {
		return 0, fmt.Errorf("failed to get stock id: %w", err)
	}

	return stockId, nil
}

func insertStockChanges(stockChange *types.StockChange) error {
	if _, err := GetConn().Exec(`INSERT INTO stock_changes(stock_id, action, quantity) VALUES($1, $2, $3)`,
		stockChange.StockId, stockChange.Action, stockChange.Quantity); err != nil {
		return fmt.Errorf("insert stock changes: %w", err)
	}

	return nil
}

func addStockItems(stockChange *types.StockChange) error {
	if _, err := GetConn().Exec(`UPDATE stock SET quantity = quantity + $1 WHERE id = $2`,
		stockChange.Quantity, stockChange.StockId); err != nil {
		return fmt.Errorf("add stock items: %w", err)
	}
	return nil
}

func removeStockItems(stockChange *types.StockChange) error {
	if _, err := GetConn().Exec(`UPDATE stock SET quantity = quantity - $1 WHERE id = $2`,
		stockChange.Quantity, stockChange.StockId); err != nil {
		return fmt.Errorf("remove stock items: %w", err)
	}
	return nil
}

func reserveStockItems(stockChange *types.StockChange) error {
	if _, err := GetConn().Exec(`UPDATE stock SET reserved = reserved + $1 WHERE id = $2`,
		stockChange.Quantity, stockChange.StockId); err != nil {
		return fmt.Errorf("reserve stock items: %w", err)
	}
	return nil
}
