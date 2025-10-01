package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"stock/types"
	"strconv"
	"strings"
	"time"

	"github.com/sethvargo/go-retry"
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
	case "add":
		return addStockItems(stockChange)
	case "remove":
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
	query := `select s.quantity, sc.quantity, s.id, s.mtime from stock s join stock_changes sc on sc.stock_id = s.id 
		where sc.id in (%s) and sc.status = 'pending'`

	backoff := retry.WithMaxRetries(retryCount, retry.NewConstant(retryDelay))
	if err := retry.Do(context.Background(), backoff, func(ctx context.Context) error {
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
				mtime    time.Time
			)

			if err := rows.Scan(&quantity, &needed, &stockID, &mtime); err != nil {
				return fmt.Errorf("get order items quantity: %w", err)
			}

			if action == StockChangeRemove && needed > quantity {
				return ErrNotEnoughItems
			}

			changes = append(changes, types.StockChange{
				StockId:  stockID,
				Quantity: needed,
				MTime:    mtime,
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
			return retry.RetryableError(fmt.Errorf(actionName+" stock items: %w", err))
		}

		return nil
	}); err != nil {
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

	operation := "+"
	if action == StockChangeRemove {
		operation = "-"
	}

	var (
		caseParts []string
		ids       []string
		args      []any
	)

	argPos := 1
	for _, ch := range changes {
		caseParts = append(caseParts,
			fmt.Sprintf("WHEN id = $%d AND mtime = $%d THEN quantity %s $%d",
				argPos, argPos+1, operation, argPos+2),
		)
		args = append(args, ch.StockId, ch.MTime, ch.Quantity)
		ids = append(ids, fmt.Sprintf("$%d", argPos))
		argPos += 3
	}

	query := fmt.Sprintf(`
		UPDATE stock
		SET quantity = CASE %s END,
		    mtime = now()
		WHERE id IN (%s)
		RETURNING id
	`, strings.Join(caseParts, " "), strings.Join(ids, ","))

	tx, err := GetConn().BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin tx for stock_changes: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.Query(query, args...)
	if err != nil {
		return fmt.Errorf("process stock_changes: %w", err)
	}
	defer rows.Close()

	updated := make(map[int64]struct{})
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan updated id: %w", err)
		}
		updated[id] = struct{}{}
	}

	// Проверяем, что все товары обновились
	if len(updated) != len(changes) {
		missing := make([]int64, 0)
		for _, ch := range changes {
			if _, ok := updated[ch.StockId]; !ok {
				missing = append(missing, ch.StockId)
			}
		}
		return fmt.Errorf("optimistic lock conflict for stock ids: %v", missing)
	}

	// Если все обновились — коммитим
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
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

func GetAllStockChanges() ([]types.StockChange, error) {
	rows, err := GetConn().Query(`select id, order_id, stock_id, action, status, quantity, error, mtime, ctime from stock_changes`)
	if err != nil {
		return nil, fmt.Errorf("get all stock changes: %w", err)
	}
	defer rows.Close()

	sc := make([]types.StockChange, 0)
	for rows.Next() {
		var (
			id, orderID, stockID int64
			action, status       string
			quantity             int64
			Error                string
			mtime, ctime         time.Time
		)

		if err := rows.Scan(&id, &orderID, &stockID, &action, &status, &quantity, &Error, &mtime, &ctime); err != nil {
			return nil, fmt.Errorf("scan all stock changes: %w", err)
		}

		sc = append(sc, types.StockChange{
			ID:       id,
			OrderID:  orderID,
			StockId:  stockID,
			Action:   action,
			Status:   status,
			Quantity: quantity,
			Error:    Error,
			MTime:    mtime,
			CTime:    ctime,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read all stock changes: %w", err)
	}

	return sc, nil
}

func GetStockChangesByOrderID(orderID int64) ([]types.StockChange, error) {
	rows, err := GetConn().Query(`select id, order_id, stock_id, action, status, quantity, error, mtime, ctime from stock_changes where order_id = $1`, orderID)
	if err != nil {
		return nil, fmt.Errorf("get stock changes: %w", err)
	}
	defer rows.Close()

	sc := make([]types.StockChange, 0)
	for rows.Next() {
		var (
			id, orderID, stockID int64
			action, status       string
			quantity             int64
			Error                string
			mtime, ctime         time.Time
		)

		if err := rows.Scan(&id, &stockID, &action, &status, &quantity, &Error, &mtime, &ctime); err != nil {
			return nil, fmt.Errorf("scan stock changes: %w", err)
		}

		sc = append(sc, types.StockChange{
			ID:       id,
			OrderID:  orderID,
			StockId:  stockID,
			Action:   action,
			Status:   status,
			Quantity: quantity,
			Error:    Error,
			MTime:    mtime,
			CTime:    ctime,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read stock changes: %w", err)
	}

	return sc, nil
}
