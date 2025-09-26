package db

import (
	"database/sql"
	"delivery/types"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.uber.org/zap"
)

func CreateCourier(userID int64) error {
	if _, err := GetConn().Exec(`insert into couriers(user_id) values($1)`, userID); err != nil {
		return fmt.Errorf("create courier: %w", err)
	}

	var rolesStr string
	if err := GetConn().QueryRow(`select roles from users where id = $1`, userID).Scan(&rolesStr); err != nil {
		return fmt.Errorf("get user roles: %w", err)
	}

	if !slices.Contains(strings.Split(rolesStr, ","), "courier") {
		if rolesStr != "" {
			rolesStr += ",courier"
		} else {
			rolesStr = "courier"
		}

		if _, err := GetConn().Exec(`update users set roles = $1 where id = $2`, rolesStr, userID); err != nil {
			return fmt.Errorf("add courier role: %w", err)
		}
	}

	zap.L().Info("courier created", zap.Int64("user_id", userID))

	return nil
}

var ErrAlreadyExistsSchedule = errors.New("schedule for today exists already")

func CreateScheduleForToday(userID int64, mask int64) error {
	var courID int64
	exists := true
	if err := GetConn().QueryRow(
		`select c.id from courier_schedule cs join couriers c on cs.courier_id = c.id where c.user_id = $1 and work_date = CURRENT_DATE`).
		Scan(&courID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			exists = false
		} else {
			return fmt.Errorf("checking existing schedule for today: %w", err)
		}
	}

	if exists {
		return ErrAlreadyExistsSchedule
	}

	if _, err := GetConn().Exec(`insert into courier_schedule(courier_id, hour_mask) values($1, $2)`, courID, mask); err != nil {
		return fmt.Errorf("create schedule for today: %w", err)
	}

	zap.L().Info("schedule for today created", zap.Int64("cour_id", courID))

	return nil
}

func GetOrdersByUserID(userID int64) ([]types.Order, error) {
	rows, err := GetConn().Query(
		`SELECT o.id, o.status, o.start_time, o.end_time
		FROM courier_reservation cr join couriers c on cr.courier_id = c.id join users on u.id = c.user_id join orders o on o.id = cr.order_id
		WHERE u.id = $1`)
	if err != nil {
		return nil, fmt.Errorf("get orders: %w", err)
	}
	defer rows.Close()

	orders := make([]types.Order, 0)
	for rows.Next() {
		var (
			orderID        int64
			orderStatus    string
			orderStartTime time.Time
			orderEndTime   sql.NullTime
		)

		if err := rows.Scan(&orderID, &orderStatus, &orderStartTime, &orderEndTime); err != nil {
			return nil, fmt.Errorf("parse orders result: %w", err)
		}

		endTimeStr := "-"
		if orderEndTime.Valid {
			endTimeStr = orderEndTime.Time.String()
		}

		orders = append(orders, types.Order{
			ID:        orderID,
			Status:    orderStatus,
			StartTime: orderStartTime.String(),
			EndTime:   endTimeStr,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading order rows: %w", err)
	}

	return orders, nil
}

func ConfirmOrderDelivery(userID, orderID int64) error {
	var rowsAffected int64
	if err := GetConn().QueryRow(
		`
		WITH updated AS (
			UPDATE courier_reservation cr
			SET status = 'delivery',
				mtime = NOW()
			FROM couriers c
			JOIN users u ON u.id = c.user_id
			JOIN orders o ON o.id = cr.order_id
			WHERE cr.courier_id = c.id
			AND u.id = $1
			AND o.id = $2
			AND o.status = 'waiting_for_cour'
			RETURNING cr.id
		)
		SELECT count(*) FROM updated;
		`, userID, orderID).Scan(&rowsAffected); err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	zap.L().Info("set order status 'delivery'", zap.Int64("order_id", orderID), zap.Int64("user_id", userID))

	return nil
}

func ConfirmOrderDelivered(userID, orderID int64) error {
	var rowsAffected int64
	if err := GetConn().QueryRow(
		`
		WITH updated AS (
			UPDATE courier_reservation cr
			SET status = 'delivered',
				mtime = NOW()
			FROM couriers c
			JOIN users u ON u.id = c.user_id
			JOIN orders o ON o.id = cr.order_id
			WHERE cr.courier_id = c.id
			AND u.id = $1
			AND o.id = $2
			AND o.status = 'delivery'
			RETURNING cr.id
		)
		SELECT count(*) FROM updated;
		`, userID, orderID).Scan(&rowsAffected); err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	zap.L().Info("set order status 'delivered'", zap.Int64("order_id", orderID), zap.Int64("user_id", userID))

	return nil
}
