package db

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
)

const (
	RevertCourReserve = iota
	CourReserve
)

var ErrUnsupportedCourReserveAction = errors.New("usupported cour_reserve action")
var ErrSlotReserved = errors.New("slot is already reserved")

func ProcessReserveCourier(courReserveID int64, action int8) error {
	if action != RevertCourReserve && action != CourReserve {
		return ErrUnsupportedCourReserveAction
	}

	var (
		courID             int64
		resMask, schedMask int64
		workDate           string
	)

	query := `
        SELECT r.courier_id, r.work_date, r.hour_mask, s.hour_mask
        FROM courier_reservation r
        JOIN courier_schedule s
          ON s.courier_id = r.courier_id AND s.work_date = r.work_date
        WHERE r.id = $1 and r.status = 'pending'
    `
	err := GetConn().QueryRow(query, courReserveID).
		Scan(&courID, &workDate, &resMask, &schedMask)
	if err != nil {
		return err
	}

	if action == CourReserve && schedMask&resMask != 0 {
		return ErrSlotReserved
	}

	if err := processReserveCourier(courID, resMask, workDate, action); err != nil {
		actionName := "reserver"
		if action == RevertCourReserve {
			actionName = "revert reserve"
		}
		return fmt.Errorf("process cour_reserve type "+actionName+": %w", err)
	}

	zap.L().Info("cour_reserve processed", zap.Int64("cour_reserve_id", courReserveID), zap.Int8("action", action))

	return nil
}

func processReserveCourier(courID int64, mask int64, workDate string, action int8) error {
	actionType := " | "
	if action == RevertCourReserve {
		actionType = " & ~"
	}

	if _, err := GetConn().Exec(
		`UPDATE courier_schedule
		SET hour_mask = hour_mask`+actionType+`$1
		WHERE courier_id = $2 AND work_date = $3
        `, mask, courID, workDate); err != nil {
		return fmt.Errorf("update courier schedule: %w", err)
	}

	zap.L().Info("courier schedule updated",
		zap.Int64("cour_id", courID),
		zap.Int64("mask", mask),
		zap.String("work_date", workDate),
		zap.Int8("action", action),
	)

	return nil
}

func ApproveReserveCourier(courReserveID int64) {
	if _, err := GetConn().Exec(
		`UPDATE courier_reservation 
		SET status = 'ok', mtime = NOW()
		WHERE id = $1
        `, courReserveID); err != nil {
		zap.L().Error("failed to approve cour_reserve", zap.Error(err))
	}

	zap.L().Info("cour_reserve approved", zap.Int64("cour_reserve_id", courReserveID))
}

func RejectReserveCourier(courReserveID int64, reason string) {
	if _, err := GetConn().Exec(
		`UPDATE courier_reservation
		SET status = 'failed', mtime = NOW(), error = $1
		WHERE id = $2
        `, reason, courReserveID); err != nil {
		zap.L().Error("failed to reject cour_reserve", zap.Error(err))
	}

	zap.L().Info("cour_reserve rejected", zap.Int64("cour_reserve_id", courReserveID), zap.String("reason", reason))
}
