package db

import (
	"database/sql"
	"fmt"

	"go.uber.org/zap"
)

func CreateCourier(name string) error {
	var courID int64
	if err := GetConn().QueryRow(`insert into couriers(name) values($1) returning id`, name).Scan(&courID); err != nil {
		return fmt.Errorf("create courier: %w", err)
	}

	if _, err := GetConn().Exec(`insert into courier_schedule(courier_id, work_date) values ($1, current_date), ($1, current_date + interval '1 day')`, courID); err != nil {
		return fmt.Errorf("create courier schedule: %w", err)
	}

	zap.L().Info("courier added", zap.Int64("cour_id", courID))

	return nil
}

func ConfirmOrderDelivery(orderID int64) error {
	var rowsAffected int64
	if err := GetConn().QueryRow(
		`update orders set status = 'delivery' where id = $1 and status = 'waiting_for_cour'`, orderID).Scan(&rowsAffected); err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	zap.L().Info("set order status 'delivery'", zap.Int64("order_id", orderID))

	return nil
}

func ConfirmOrderDelivered(orderID int64) error {
	var rowsAffected int64
	if err := GetConn().QueryRow(
		`update orders set status = 'delivered', end_time = now() where id = $1 and status = 'delivery'`, orderID).Scan(&rowsAffected); err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	zap.L().Info("set order status 'delivered'", zap.Int64("order_id", orderID))

	return nil
}
