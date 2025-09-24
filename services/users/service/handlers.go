package service

import (
	"encoding/json"
	"errors"
	"users/db"
	"users/types"

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

func handleUser(ctx *fasthttp.RequestCtx, userId int64) {
	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		getUser(ctx, userId)
	case fasthttp.MethodPost:
		updateUser(ctx, userId)
	case fasthttp.MethodDelete:
		deleteUser(ctx, userId)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

var (
	ErrBadInput   = errors.New("bad input")
	ErrDeleteUser = errors.New("delete user error")
	ErrUpdateUser = errors.New("update user error")
	ErrGetUser    = errors.New("get user error")
)

// deleteUser godoc
//
//	@Summary		delete user
//	@Description	delete user
//	@Tags			users
//	@Success		204	{object}	nil
//	@Failure		400	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/user/ [delete]
func deleteUser(ctx *fasthttp.RequestCtx, userId int64) {
	if err := db.DeleteUser(userId); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrDeleteUser, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

// getUser godoc
//
//	@Summary		get user
//	@Description	get user
//	@Tags			users
//	@Produce		json
//	@Success		200	{object}	types.User
//	@Failure		400	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/user/ [get]
func getUser(ctx *fasthttp.RequestCtx, userId int64) {
	user, err := db.GetUser(userId)
	if err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrGetUser, fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(user)
}

// updateUser godoc
//
//	@Summary		update user
//	@Description	update user
//	@Tags			users
//	@Accept			json
//	@Success		200	{object}	nil
//	@Failure		400	{object}	types.HTTPError
//	@Failure		404	{object}	types.HTTPError
//	@Failure		405	{object}	types.HTTPError
//	@Failure		500	{object}	types.HTTPError
//	@Router			/user/ [post]
func updateUser(ctx *fasthttp.RequestCtx, userId int64) {
	var user types.User
	if err := json.Unmarshal(ctx.Request.Body(), &user); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrBadInput, fasthttp.StatusBadRequest)
		return
	}

	user.Id = userId
	if err := db.UpdateUser(&user); err != nil {
		zap.L().Error(err.Error())
		handleError(ctx, ErrUpdateUser, fasthttp.StatusBadRequest)
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
