package db

import "stock/types"

func GetItems() ([]types.Item, error) {
	rows, err := GetConn().Query("SELECT i.id, i.name, i.description, i.price, s.quantity FROM items i join stock s on s.item_id = i.id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.Item
	for rows.Next() {
		var item types.Item
		if err := rows.Scan(&item.Id, &item.Name, &item.Description, &item.Price, &item.Quantity); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func AddItem(item *types.Item) error {
	res := GetConn().QueryRow(`INSERT INTO items (name, description, price, mtime) VALUES ($1, $2, $3, NOW()) RETURNING id`, item.Name, item.Description, item.Price)

	if err := res.Scan(&item.Id); err != nil {
		return err
	}

	if _, err := GetConn().Exec(`INSERT INTO stock(item_id) VALUES($1)`, item.Id); err != nil {
		// Rollback
		GetConn().Exec(`DELETE FROM ITEMS WHERE id = $1`, item.Id)
		return err
	}

	return nil
}

func UpdateItem(item *types.Item) error {
	_, err := GetConn().Exec(`UPDATE items SET name = $1, description = $2, price = $3, mtime = NOW() WHERE id = $4`, item.Name, item.Description, item.Price, item.Id)
	return err
}
