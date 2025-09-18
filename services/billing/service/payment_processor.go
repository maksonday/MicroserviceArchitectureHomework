package service

import (
	"billing/db"
	"billing/types"
	"encoding/json"

	"github.com/valyala/fasthttp"
)

func payment(ctx *fasthttp.RequestCtx, userId int64) {
	var payment types.Payment
	if err := json.Unmarshal(ctx.Request.Body(), &payment); err != nil {
		handleError(ctx, err, fasthttp.StatusBadRequest)
		return
	}

	if err := db.ProcessPayment(userId, payment.Amount); err != nil {
		handleError(ctx, err, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}
