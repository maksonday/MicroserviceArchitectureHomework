package service

import (
	"billing/db"
	"billing/types"
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
	ErrBadInput      = errors.New("bad input")
	ErrGetBalance    = errors.New("get balance error")
	ErrAddMoney      = errors.New("deposit money error")
	ErrCreateAccount = errors.New("create account error")
	ErrInternal      = errors.New("internal error, try again lates")
)

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
		zap.L().Error(err.Error())
		handleError(ctx, ErrCreateAccount, fasthttp.StatusBadRequest)
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
		zap.L().Error(err.Error())
		handleError(ctx, ErrGetBalance, fasthttp.StatusBadRequest)
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
		zap.L().Error(err.Error())
		handleError(ctx, ErrBadInput, fasthttp.StatusBadRequest)
		return
	}

	if err := db.AddMoney(userId, deposit.Amount); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrAddMoney, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

// getPayments godoc
//
//	@Summary		get_payments
//	@Description	get_payments
//	@Tags			billing
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	[]types.Payment
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/get_payments [post]
func getPayments(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	var req types.PaymentsListRequest
	if err := json.Unmarshal(ctx.Request.Body(), &req); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrBadInput, fasthttp.StatusBadRequest)
		return
	}

	payments, err := db.GetPaymentsByOrderID(req.OrderID)
	if err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrInternal, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(payments)
}

func handleError(ctx *fasthttp.RequestCtx, err error, status int) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(types.HTTPError{
		Error: err.Error(),
	})
}
