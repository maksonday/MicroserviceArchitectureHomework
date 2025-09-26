package db

import "fmt"

func CreateCourReserve(orderID int64) (int64, error) {
	var mask int64
	if err := GetConn().QueryRow(`select hour_mask, address from orders where id = $1`, orderID).Scan(&mask); err != nil {
		return 0, fmt.Errorf("get order hour_mask: %w", err)
	}

	var courReserveID int64
	if err := GetConn().QueryRow(
		`insert into courier_reservation(order_id, action, hour_mask) values($1, 'reserve', $2) returning id`,
		orderID, mask).Scan(&courReserveID); err != nil {
		return 0, fmt.Errorf("create cour_reserve: %w", err)
	}

	return courReserveID, nil
}

func buildRevertCourReserve(courReserveID int64) (int64, int64, error) {
	var (
		orderID int64
		mask    int64
	)

	if err := GetConn().QueryRow(`select order_id, hour_mask from courier_reservation where id = $1 and action = 'reserve'`, courReserveID).
		Scan(&orderID, &mask); err != nil {
		return 0, 0, err
	}

	return orderID, mask, nil
}

func RevertCourReserve(courReserveID int64) (int64, error) {
	orderID, mask, err := buildRevertCourReserve(courReserveID)
	if err != nil {
		return 0, fmt.Errorf("build revert cour_reserve: %w", err)
	}

	var newID int64
	if err := GetConn().QueryRow(
		`insert into courier_reservation(order_id, hour_mask, action) values ($1, $2, 'revert_reserve') returning id`, orderID, mask).
		Scan(&newID); err != nil {
		return 0, err
	}

	return newID, nil
}
