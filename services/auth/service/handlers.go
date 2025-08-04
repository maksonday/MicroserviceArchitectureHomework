package service

import (
	"encoding/json"
	"miniapp/db"
	"miniapp/types"
	"strconv"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func healthCheckHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.WriteString(`{"status":"OK"}`)
}

func handleUser(ctx *fasthttp.RequestCtx, userIdStr string) {
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		handleError(ctx, err)
		return
	}

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

func deleteUser(ctx *fasthttp.RequestCtx, userId int64) {
	if err := db.DeleteUser(userId); err != nil {
		handleError(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

func getUser(ctx *fasthttp.RequestCtx, userId int64) {
	user, err := db.GetUser(userId)
	if err != nil {
		handleError(ctx, err)
		return
	}

	ctx.SetContentType("application/json")
	response, err := json.Marshal(user)
	if err != nil {
		handleError(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Write(response)
}

func updateUser(ctx *fasthttp.RequestCtx, userId int64) {
	var user types.User
	if err := json.Unmarshal(ctx.Request.Body(), &user); err != nil {
		handleError(ctx, err)
		return
	}

	user.Id = userId
	if err := db.UpdateUser(&user); err != nil {
		handleError(ctx, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func handleError(ctx *fasthttp.RequestCtx, err error) {
	ctx.SetStatusCode(fasthttp.StatusBadRequest)
	ctx.SetContentType("application/json")
	response := map[string]string{"error": err.Error()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		zap.L().Error("failed to marshal error response: " + err.Error())
		return
	}
	ctx.Write(jsonResponse)
}
