package service

import (
	"delivery/db"
	"delivery/types"
	"encoding/json"
	"errors"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

const (
	blAtKeyPrefix = "bl:at:"
	blRtKeyPrefix = "bl:rt:"
)

func healthCheckHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.WriteString(`{"status":"OK"}`)
}

var (
	ErrBadInput = errors.New("bad input")
	ErrInternal = errors.New("internal error, try again later")
)

func applyWork(ctx *fasthttp.RequestCtx, userID int64) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	if err := db.CreateCourier(userID); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func createScheduleForToday(ctx *fasthttp.RequestCtx, userID int64) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var schedule types.Schedule
	if err := json.Unmarshal(ctx.Request.Body(), &schedule); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	if err := db.CreateScheduleForToday(userID, schedule.Mask); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func getOrders(ctx *fasthttp.RequestCtx, userID int64) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	orders, err := db.GetOrdersByUserID(userID)
	if err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(orders)
}

func confirmOrderDelivery(ctx *fasthttp.RequestCtx, userID int64) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var order types.Order
	if err := json.Unmarshal(ctx.Request.Body(), &order); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	if err := db.ConfirmOrderDelivery(userID, order.ID); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	go NotifyUser(order.ID, OrderStatusDelivery)

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func confirmOrderDelivered(ctx *fasthttp.RequestCtx, userID int64) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var order types.Order
	if err := json.Unmarshal(ctx.Request.Body(), &order); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	if err := db.ConfirmOrderDelivered(userID, order.ID); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	go NotifyUser(order.ID, OrderStatusDelivered)

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func handleError(ctx *fasthttp.RequestCtx, err error, status int) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(types.HTTPError{
		Error: err.Error(),
	})
}
