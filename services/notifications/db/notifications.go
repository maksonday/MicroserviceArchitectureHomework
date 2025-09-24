package db

import "notifications/types"

func GetNotificationsByUserID(userID int64) ([]types.Notification, error) {
	rows, err := GetConn().Query(`select message from notifications where user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]types.Notification, 0)
	for rows.Next() {
		var message string
		if err := rows.Scan(&message); err != nil {
			return nil, err
		}

		messages = append(messages, types.Notification{
			Message: message,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
