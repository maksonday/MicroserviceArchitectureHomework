package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"order/db"
	"order/types"

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
	ErrCreateOrder           = errors.New("create order error")
	ErrCalculateDeliveryTime = errors.New("calculate delivery time error")
)

// create_order godoc
//
//	@Summary		create order
//	@Description	create order
//	@Tags			order
//	@Accept			json
//	@Produce        json
//	@Success		200	{object}	types.CreateOrderResponse
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/create_order [post]
func handleCreateOrder(userID int64, ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var order types.Order
	if err := json.Unmarshal(ctx.Request.Body(), &order); err != nil {
		handleError(ctx, err, fasthttp.StatusBadRequest)
		return
	}

	mask, err := calculateOrderMaskFromAddress(order.Address)
	if err != nil {
		zap.L().Error(fmt.Errorf("calculate delivery time: %w", err).Error())
		handleError(ctx, ErrCreateOrder, fasthttp.StatusBadRequest)
		return
	}

	orderID, err := db.CreateOrder(userID, mask, &order)
	if err != nil {
		zap.L().Error(fmt.Errorf("create order: %w", err).Error())
		handleError(ctx, ErrCreateOrder, fasthttp.StatusBadRequest)
		return
	}

	order.ID = orderID

	go postCreateOrder(&order)

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(types.CreateOrderResponse{ID: orderID})
}

// TODO replace with service call
func calculateOrderMaskFromAddress(_ string) (int64, error) {
	return 1 << 14, nil
}

func postCreateOrder(order *types.Order) {
	var (
		stockChangeIDs []int64
		err            error
	)

	if stockChangeIDs, err = db.CreateStockChanges(order.ID, order.Items); err != nil {
		zap.L().Error("create stock changes", zap.Error(err))
		db.RejectOrder(order.ID)
		return
	}

	GetStockProcessor().AddMessage(&StockChangeMessage{
		StockChangeIDs: stockChangeIDs,
		OrderID:        order.ID,
		Status:         PaymentStatusPending,
		Action:         StockRemove,
	})
}

var ErrGetOrders = errors.New("get orders error")

// get_orders godoc
//
//	@Summary		get orders
//	@Description	get orders
//	@Tags			order
//	@Produce		json
//	@Success		200	{object}	[]types.Order
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/get_orders [get]
func handleGetOrders(userID int64, ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	orders, err := db.GetOrders(userID)
	if err != nil {
		zap.L().Error(fmt.Errorf("get orders: %w", err).Error())
		handleError(ctx, ErrGetOrders, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(orders)
}

func handleError(ctx *fasthttp.RequestCtx, err error, status int) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(types.HTTPError{
		Error: err.Error(),
	})
}
