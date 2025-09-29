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

func addNewCourier(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var cour types.Courier
	if err := json.Unmarshal(ctx.Request.Body(), &cour); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrBadInput, fasthttp.StatusBadRequest)
		return
	}

	if err := db.CreateCourier(cour.Name); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func confirmOrderDelivery(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var order types.Order
	if err := json.Unmarshal(ctx.Request.Body(), &order); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrBadInput, fasthttp.StatusBadRequest)
		return
	}

	if err := db.ConfirmOrderDelivery(order.OrderID); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	go NotifyUser(order.OrderID, OrderStatusDelivery)

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func confirmOrderDelivered(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var order types.Order
	if err := json.Unmarshal(ctx.Request.Body(), &order); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrBadInput, fasthttp.StatusBadRequest)
		return
	}

	if err := db.ConfirmOrderDelivered(order.OrderID); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	go NotifyUser(order.OrderID, OrderStatusDelivered)

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func getCourReservations(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var req types.CourReserveListRequest
	if err := json.Unmarshal(ctx.Request.Body(), &req); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrBadInput, fasthttp.StatusBadRequest)
		return
	}

	cr, err := db.GetCourReservationsByOrderID(req.OrderID)
	if err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(cr)
}

func handleError(ctx *fasthttp.RequestCtx, err error, status int) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(types.HTTPError{
		Error: err.Error(),
	})
}
