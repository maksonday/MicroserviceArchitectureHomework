package db

func GetUserByOrderID(orderID int64) (int64, error) {
	var userID int64
	if err := GetConn().QueryRow(`select user_id from orders where id = $1`, orderID).Scan(&userID); err != nil {
		return 0, err
	}

	return userID, nil
}
