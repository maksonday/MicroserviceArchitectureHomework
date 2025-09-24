package db

import (
	"database/sql"
	"errors"
	"fmt"
	"stock/types"
	"strconv"
	"strings"

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

var (
	ErrNotEnoughItems = errors.New("not enough items in stock")
)

func ProcessStockChangesAsync(stockChangeIDs []int64, action int8) error {
	if action != StockChangeAdd && action != StockChangeRemove {
		return ErrUnsupportedStockChangeAction
	}

	changesStr := changesToStr(stockChangeIDs)
	query := `select s.quantity, sc.quantity, s.id needed from stock s join stock_changes sc on sc.stock_id = s.id 
		where sc.id in (%s) and sc.status = 'pending'`

	rows, err := GetConn().Query(fmt.Sprintf(query, strings.Join(changesStr, ",")))
	if err != nil {
		return fmt.Errorf("get stock_changes: %w", err)
	}
	defer rows.Close()

	changes := make([]types.StockChange, 0, len(stockChangeIDs))
	for rows.Next() {
		var (
			quantity int64
			needed   int64
			stockID  int64
		)

		if err := rows.Scan(&quantity, &needed, &stockID); err != nil {
			return fmt.Errorf("get order items quantity: %w", err)
		}

		if needed > quantity {
			return ErrNotEnoughItems
		}

		changes = append(changes, types.StockChange{
			StockId:  stockID,
			Quantity: needed,
		})
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows scan: %w", err)
	}

	if len(changes) == 0 {
		return sql.ErrNoRows
	}

	// уменьшаем кол-во вещей на складе, если все ок, иначе возвращаем обратно
	if err := processStockChanges(changes, action); err != nil {
		actionName := "remove"
		if action == StockChangeAdd {
			actionName = "add"
		}
		err = fmt.Errorf(actionName+" stock items: %w", err)
		RejectStockChanges(strings.Join(changesStr, ","), err.Error())
		return err
	}

	ApproveStockChanges(strings.Join(changesStr, ","))

	return nil
}

func changesToStr(changes []int64) []string {
	changesStr := make([]string, 0, len(changes))
	for _, id := range changes {
		changesStr = append(changesStr, strconv.FormatInt(id, 10))
	}

	return changesStr
}

func processStockChanges(changes []types.StockChange, action int8) error {
	query := `update stock set quantity = case %s end`
	values := make([]string, 0, len(changes))
	switch action {
	case StockChangeAdd:
		for _, change := range changes {
			values = append(values, fmt.Sprintf(`when id = %d then quantity = quantity + %d`, change.StockId, change.Quantity))
		}
	case StockChangeRemove:
		for _, change := range changes {
			values = append(values, fmt.Sprintf(`when id = %d then quantity = quantity - %d`, change.StockId, change.Quantity))
		}
	default:
		return ErrUnsupportedStockChangeAction
	}

	if _, err := GetConn().Exec(fmt.Sprintf(query, strings.Join(values, "\t"))); err != nil {
		return fmt.Errorf("process stock_changes: %w", err)
	}

	return nil
}

func ApproveStockChanges(changes string) {
	if _, err := GetConn().Exec(
		fmt.Sprintf(`update stock_changes set status = 'ok', mtime = NOW() where id in (%s)`, changes)); err != nil {
		zap.L().Error("failed to approve stock remove", zap.Error(err))
	}
}

func RejectStockChanges(changes, reason string) {
	if _, err := GetConn().Exec(
		fmt.Sprintf(`update stock_changes set status = 'failed', error = %s, mtime = NOW() where id in (%s)'`,
			reason, changes)); err != nil {
		zap.L().Error("failed to reject stock_change", zap.Error(err))
	}
}
