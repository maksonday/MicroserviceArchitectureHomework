package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"notifications/db"
	"notifications/types"

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

var ErrGetNotifications = errors.New("get notifications error")

// get_notifications godoc
//
//	@Summary		get notifications
//	@Description	get notifications
//	@Tags			notifications
//	@Produce		json
//	@Success		200	{object}	[]types.Notification
//	@Failure		400	{object}	types.HTTPError
//	@Failure		401	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/get_notifications [get]
func handleGetNotifications(userID int64, ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.Error("method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	messages, err := db.GetNotificationsByUserID(userID)
	if err != nil {
		zap.L().Error(fmt.Errorf("get notifications: %w", err).Error())
		handleError(ctx, ErrGetNotifications, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(messages)
}

func handleError(ctx *fasthttp.RequestCtx, err error, status int) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(types.HTTPError{
		Error: err.Error(),
	})
}
