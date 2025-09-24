package service

import (
	"encoding/json"
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

	orderID, err := db.CreateOrder(userID, &order)
	if err != nil {
		handleError(ctx, fmt.Errorf("create order: %w", err), fasthttp.StatusBadRequest)
		return
	}

	order.ID = orderID

	go postCreateOrder(&order)

	ctx.SetStatusCode(fasthttp.StatusCreated)
}

func postCreateOrder(order *types.Order) {
	var (
		stockChangeIDs []int64
		err            error
	)

	if stockChangeIDs, err = db.CreateStockChanges(order.ID, order.Items); err != nil {
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

func handleGetOrder(userID int64, ctx *fasthttp.RequestCtx) {

}

func handleError(ctx *fasthttp.RequestCtx, err error, status int) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	zap.L().Error(err.Error())
	json.NewEncoder(ctx).Encode(map[string]string{
		"error": err.Error(),
	})
}
