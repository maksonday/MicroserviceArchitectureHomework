package db

import (
	"notifications/types"

	"go.uber.org/zap"
)

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

func CreateNotification(userID int64, msg string) {
	if _, err := GetConn().Exec(`insert into notifications(user_id, message) values($1, $2)`, userID, msg); err != nil {
		zap.L().Error("failed to insert notification", zap.Error(err))
	}
}
