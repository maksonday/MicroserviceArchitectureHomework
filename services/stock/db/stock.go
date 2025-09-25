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
	query := `select s.quantity, sc.quantity, s.id from stock s join stock_changes sc on sc.stock_id = s.id 
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
		return err
	}

	zap.L().Info("processed stock changes", zap.Int64s("stock_change_ids", stockChangeIDs), zap.Int8("action", action))

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
	if action != StockChangeAdd && action != StockChangeRemove {
		return ErrUnsupportedStockChangeAction
	}

	idsStr := make([]string, 0, len(changes))
	for _, ch := range changes {
		idsStr = append(idsStr, strconv.FormatInt(ch.StockId, 10))
	}

	query := `update stock set quantity = (case %s end) where id in (%s)`
	values := make([]string, 0, len(changes))
	operation := "+"
	if action == StockChangeRemove {
		operation = "-"
	}

	for _, change := range changes {
		values = append(values, fmt.Sprintf(`when id = %d then quantity %s %d`, change.StockId, operation, change.Quantity))
	}

	if _, err := GetConn().Exec(fmt.Sprintf(query, strings.Join(values, "\t"), strings.Join(idsStr, ","))); err != nil {
		return fmt.Errorf("process stock_changes: %w", err)
	}

	zap.L().Info("updated stock", zap.Any("stock_changes", changes))

	return nil
}

func ApproveStockChanges(stockChangeIDs []int64) {
	changes := changesToStr(stockChangeIDs)
	if _, err := GetConn().Exec(
		fmt.Sprintf(`update stock_changes set status = 'ok', mtime = NOW() where id in (%s)`, strings.Join(changes, ","))); err != nil {
		zap.L().Error("failed to approve stock remove", zap.Error(err))
		return
	}

	zap.L().Info("approved stock changes", zap.Int64s("stock_change_ids", stockChangeIDs))
}

func RejectStockChanges(stockChangeIDs []int64, reason string) {
	changes := changesToStr(stockChangeIDs)
	query := fmt.Sprintf(
		`update stock_changes set status = 'failed', error = $1, mtime = NOW() where id in (%s)`, strings.Join(changes, ","))
	if _, err := GetConn().Exec(query, reason); err != nil {
		zap.L().Error("failed to reject stock_changes", zap.Error(err), zap.String("query", query))
		return
	}

	zap.L().Info("rejected stock changes", zap.Int64s("stock_change_ids", stockChangeIDs), zap.String("reason", reason))
}
