package db

import (
	"errors"
	"fmt"
	"stock/types"

	"go.uber.org/zap"
)

const (
	StockChangeAdd = iota
	StockChangeRemove
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
	switch stockChange.Action {
	case StockChangeAdd:
		return addStockItems(stockChange)
	case StockChangeRemove:
		return removeStockItems(stockChange)
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

var ErrNotEnoughItems = errors.New("not enough items in stock")

func ProcessStockChangeAsync(stockChangeID, stockID, orderID int64) error {
	var (
		quantity int64
		needed   int64
	)
	if err := GetConn().QueryRow(
		`select s.quantity, sc.quantity needed from stock s join stock_changes sc on sc.stock_id = s.id 
		where s.id = $1 and sc.id = $2 and sc.order_id = $3 and sc.status = 'pending'`,
		stockID, stockChangeID, orderID).Scan(&quantity, &needed); err != nil {
		return fmt.Errorf("get order items quantity: %w", err)
	}

	if needed > quantity {
		return ErrNotEnoughItems
	}

	if err := removeStockItems(&types.StockChange{StockId: stockID, Quantity: needed}); err != nil {
		return fmt.Errorf("remove stock items: %w", err)
	}

	if _, err := GetConn().Exec(
		`update stock_changes set status = 'ok', mtime = NOW() where id = $1 and status = 'pending'`, stockChangeID); err != nil {
		addStockItems(&types.StockChange{StockId: stockID, Quantity: needed})
		return fmt.Errorf("update stock_change status: %w", err)
	}

	return nil
}

func RejectStockChange(stockChangeID int64, reason string) {
	if _, err := GetConn().Exec(
		`update stock_changes set status = 'failed', error = $1, mtime = NOW() where id = $2 and status = 'pending'`,
		reason, stockChangeID); err != nil {
		zap.L().Error("failed to reject stock_change", zap.Error(err))
	}
}
