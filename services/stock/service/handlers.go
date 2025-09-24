package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"stock/db"
	"stock/types"

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

func handleGetItems(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	items, err := db.GetItems()
	if err != nil {
		handleError(ctx, fmt.Errorf("get items: %w", err), fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	if err := json.NewEncoder(ctx).Encode(items); err != nil {
		handleError(ctx, fmt.Errorf("encode items: %w", err), fasthttp.StatusBadRequest)
		return
	}
}

var (
	ErrNoItemName  = errors.New("no item name")
	ErrNoItemDesc  = errors.New("no item description")
	ErrNoItemPrice = errors.New("item price is not provided or less than 0.01")
	ErrBadInput    = errors.New("bad input")
	ErrAddItem     = errors.New("add item error")
	ErrUpdateItem  = errors.New("update item error")
	ErrUpdateStock = errors.New("update stock error")
)

func validateItem(item *types.Item) error {
	switch {
	case len(item.Name) == 0:
		return ErrNoItemName
	case len(item.Description) == 0:
		return ErrNoItemDesc
	case item.Price < 0.01:
		return ErrNoItemPrice
	default:
		return nil
	}
}

// add_item godoc
//
//	@Summary		add item
//	@Description	add item
//	@Tags			stock
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	types.Item
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/add_item [post]
func handleAddItem(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var item types.Item
	if err := json.Unmarshal(ctx.Request.Body(), &item); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrBadInput, fasthttp.StatusBadRequest)
		return
	}

	if err := validateItem(&item); err != nil {
		zap.L().Error("validate item", zap.Error(err))
		handleError(ctx, err, fasthttp.StatusBadRequest)
		return
	}

	if err := db.AddItem(&item); err != nil {
		zap.L().Error(fmt.Errorf("add item: %w", err).Error())
		handleError(ctx, ErrAddItem, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(item)
}

// update_item godoc
//
//	@Summary		update_item
//	@Description	update_item
//	@Tags			stock
//	@Accept			json
//	@Success		200	{object}	nil
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/update_item [post]
func handleUpdateItem(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var item types.Item
	if err := json.Unmarshal(ctx.Request.Body(), &item); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrBadInput, fasthttp.StatusBadRequest)
		return
	}

	if err := validateItem(&item); err != nil {
		zap.L().Error("validate item", zap.Error(err))
		handleError(ctx, err, fasthttp.StatusBadRequest)
		return
	}

	if err := db.UpdateItem(&item); err != nil {
		zap.L().Error(fmt.Errorf("update item: %w", err).Error())
		handleError(ctx, ErrUpdateItem, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

// stock_change godoc
//
//	@Summary		stock_change
//	@Description	stock_change
//	@Tags			stock
//	@Accept			json
//	@Success		200	{object}	nil
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/stock_change [post]
func handleStockChange(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var stockChange types.StockChange
	if err := json.Unmarshal(ctx.Request.Body(), &stockChange); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrBadInput, fasthttp.StatusBadRequest)
		return
	}

	if err := db.ProcessStockChange(&stockChange); err != nil {
		zap.L().Error(fmt.Errorf("update stock change: %w", err).Error())
		handleError(ctx, ErrUpdateStock, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func handleError(ctx *fasthttp.RequestCtx, err error, status int) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(types.HTTPError{
		Error: err.Error(),
	})
}
