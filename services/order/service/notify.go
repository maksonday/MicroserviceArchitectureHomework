package service

import "go.uber.org/zap"

const (
	OrderStatusPending = iota
	OrderStatusApproved
	OrderStatusCanceled
	OrderStatusDelivered
)

func NotifyUser(orderID int64, status int8) {
	zap.L().Sugar().Infof("notify user: orderID %d, status %d", orderID, status)
}
