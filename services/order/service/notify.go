package service

import (
	"fmt"
	"order/db"

	"go.uber.org/zap"
)

const (
	OrderStatusPending = iota
	OrderStatusApproved
	OrderStatusCanceled
	OrderStatusDelivery
	OrderStatusDelivered
)

func NotifyUser(orderID int64, status int8) {
	userID, err := db.GetUserByOrderID(orderID)
	if err != nil {
		zap.L().Error("get user by order id", zap.Error(err))
	}

	var statusName string
	switch status {
	case OrderStatusPending:
		statusName = "pending"
	case OrderStatusApproved:
		statusName = "approved"
	case OrderStatusCanceled:
		statusName = "canceled"
	case OrderStatusDelivery:
		statusName = "delivery"
	case OrderStatusDelivered:
		statusName = "delivered"
	}

	GetNotificationsProcessor().AddMessage(&NotificationMessage{
		UserID:  userID,
		Message: fmt.Sprintf("Order #%d status: %s", orderID, statusName),
		OrderID: orderID,
	})

	zap.L().Sugar().Infof("notify user: orderID %d, status %s", orderID, statusName)
}
