package db

import "delivery/metrics"

func GetUserByOrderID(orderID int64) (int64, error) {
	var userID int64
	if err := GetConnSlave().QueryRow(`select user_id from orders where id = $1`, orderID).Scan(&userID); err != nil {
		metrics.DBReadErrorsCount.Inc()
		return 0, err
	}

	return userID, nil
}
