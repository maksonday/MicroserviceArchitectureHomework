package db

import (
	"errors"
	"fmt"
	"order/types"
	"strconv"
	"strings"
)

var ErrMissingItemsInStock = errors.New("missing items in stock")

func CreateStockChanges(orderID int64, items []types.Item) ([]int64, error) {
	stockChanges, err := buildStockChanges(orderID, items)
	if err != nil {
		return nil, fmt.Errorf("build stock_changes: %w", err)
	}

	query := `insert into stock_changes(order_id, stock_id, action, quantity) values %s returning id`
	values := make([]string, 0, len(stockChanges))
	for _, sc := range stockChanges {
		values = append(values, fmt.Sprintf("(%d, %d, 'remove', %d)", sc.OrderID, sc.StockID, sc.Quantity))
	}

	rows, err := GetConn().Query(fmt.Sprintf(query, strings.Join(values, ",")))
	if err != nil {
		return nil, fmt.Errorf("insert stock_changes: %w", err)
	}
	defer rows.Close()

	stockChangesIDs := make([]int64, 0, len(stockChanges))
	for rows.Next() {
		var stockChangeID int64
		if err := rows.Scan(&stockChangeID); err != nil {
			return nil, fmt.Errorf("scan stock_change id: %w", err)
		}

		stockChangesIDs = append(stockChangesIDs, stockChangeID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("insert stock_changes: %w", err)
	}

	return stockChangesIDs, nil
}

func buildStockChanges(orderID int64, items []types.Item) ([]types.Item, error) {
	itemIDstr := make([]string, 0, len(items))
	itemsMap := make(map[int64]int64, len(items))
	for _, item := range items {
		itemIDstr = append(itemIDstr, strconv.FormatInt(item.Id, 10))
		itemsMap[item.Id] = item.Quantity
	}

	rows, err := GetConn().Query(fmt.Sprintf(`select id, item_id from stock where item_id in (%s)`, strings.Join(itemIDstr, ",")))
	if err != nil {
		return nil, fmt.Errorf("get items from stock: %w", err)
	}
	defer rows.Close()

	stockChanges := make([]types.Item, 0, len(items))
	for rows.Next() {
		var stockId, itemId int64
		if err := rows.Scan(&stockId, &itemId); err != nil {
			return nil, fmt.Errorf("scan stock ids: %w", err)
		}

		if _, ok := itemsMap[itemId]; !ok {
			return nil, fmt.Errorf("missing item id in stock: %w", err)
		}
		stockChanges = append(stockChanges, types.Item{
			StockID:  stockId,
			OrderID:  orderID,
			Quantity: itemsMap[itemId],
			Id:       itemId,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("process rows stock_changes: %w", err)
	}

	if len(stockChanges) != len(items) {
		return nil, ErrMissingItemsInStock
	}

	return stockChanges, nil
}

func RejectStockChanges(stockChangeIDs []int64) {
	idsStr := make([]string, 0, len(stockChangeIDs))
	for _, id := range stockChangeIDs {
		idsStr = append(idsStr, strconv.FormatInt(id, 10))
	}

	GetConn().Exec(fmt.Sprintf(`update stock_changes set status = 'failed' where id in (%s)`, strings.Join(idsStr, ",")))
}
