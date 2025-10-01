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

// add_courier godoc
//
//	@Summary		add_courier
//	@Description	add_courier
//	@Tags			delivery
//	@Accept			json
//	@Success		200	{object}	nil
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/add_courier [post]
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

// confirm_delivered godoc
//
//	@Summary		confirm_delivered
//	@Description	confirm_delivered
//	@Tags			delivery
//	@Accept			json
//	@Success		200	{object}	nil
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/confirm_delivered [post]
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

// get_courier_reservations godoc
//
//	@Summary		get_courier_reservations
//	@Description	get_courier_reservations
//	@Tags			delivery
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	[]types.CourierReservation
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/get_courier_reservations [post]
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

// get_all_courier_reservations godoc
//
//	@Summary		get_all_courier_reservations
//	@Description	get_all_courier_reservations
//	@Tags			delivery
//	@Produce		json
//	@Success		200	{object}	[]types.CourierReservation
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/get_all_courier_reservations [get]
func getAllCourReservations(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	cr, err := db.GetAllCourReservations()
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
