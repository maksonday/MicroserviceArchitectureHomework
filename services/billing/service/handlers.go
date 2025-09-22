package service

import (
	"billing/db"
	"billing/types"
	"encoding/json"

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

// create_account godoc
//
//	@Summary		create account
//	@Description	create account
//	@Tags			billing
//	@Success		201	{object}	nil
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/create_account [get]
func createAccount(ctx *fasthttp.RequestCtx, userId int64) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	if _, err := db.CreateAccount(userId); err != nil {
		handleError(ctx, err, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusCreated)
}

// getBalance godoc
//
//	@Summary		get balance
//	@Description	get balance
//	@Tags			billing
//	@Produce		json
//	@Success		200	{object}	types.BalanceResponse
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/get_balance [get]
func getBalance(ctx *fasthttp.RequestCtx, userId int64) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	balance, err := db.GetBalance(userId)
	if err != nil {
		handleError(ctx, err, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(types.BalanceResponse{Balance: balance})
}

// addMoney godoc
//
//	@Summary		add money
//	@Description	add money
//	@Tags			billing
//	@Accept			json
//	@Success		200	{object}	nil
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/add_money [post]
func addMoney(ctx *fasthttp.RequestCtx, userId int64) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var deposit types.Deposit
	if err := json.Unmarshal(ctx.Request.Body(), &deposit); err != nil {
		handleError(ctx, err, fasthttp.StatusBadRequest)
		return
	}

	if err := db.AddMoney(userId, deposit.Amount); err != nil {
		handleError(ctx, err, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func handleError(ctx *fasthttp.RequestCtx, err error, status int) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	zap.L().Error(err.Error())
	json.NewEncoder(ctx).Encode(types.HTTPError{
		Error: err.Error(),
	})
}
