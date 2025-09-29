package db

import (
	"delivery/types"
	"errors"
	"fmt"
	"time"

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
		actionName := "reserve"
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
		fmt.Sprintf(`UPDATE courier_schedule
		SET hour_mask = hour_mask`+actionType+`%d
		WHERE courier_id = $1 AND work_date = $2
        `, mask), courID, workDate); err != nil {
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

func GetCourReservationsByOrderID(orderID int64) ([]types.CourierReservation, error) {
	rows, err := GetConn().Query(`select id, order_id, courier_id, action, status, work_date, hour_mask, error, ctime, mtime from courier_reservation where order_id = $1`, orderID)
	if err != nil {
		return nil, fmt.Errorf("get cour reservation list: %w", err)
	}
	defer rows.Close()

	cr := make([]types.CourierReservation, 0)
	for rows.Next() {
		var (
			id, orderID, courID int64
			action, status      string
			workDate            time.Time
			hourMask            int64
			Error               string
			ctime, mtime        time.Time
		)

		if err := rows.Scan(&id, &orderID, &courID, &action, &status, &workDate, &hourMask, &Error, &ctime, &mtime); err != nil {
			return nil, fmt.Errorf("scan cour reservations: %w", err)
		}

		hourIsSet := 0
		for i := 0; i < 24; i++ {
			if (hourMask>>i)&1 != 0 {
				hourIsSet = i
				break
			}
		}

		cr = append(cr, types.CourierReservation{
			ID:        id,
			OrderID:   orderID,
			CourID:    courID,
			Action:    action,
			Status:    status,
			Error:     Error,
			CTime:     ctime,
			MTime:     mtime,
			StartTime: workDate.Add(time.Hour * time.Duration(hourIsSet)),
			EndTime:   workDate.Add(time.Hour * time.Duration(hourIsSet+1)),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read cour reservation: %w", err)
	}

	return cr, nil
}
