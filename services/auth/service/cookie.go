package service

import (
	"time"

	"github.com/valyala/fasthttp"
)

const refreshCookieName = "refresh_token"

func retrieveRefreshTokenFromCtx(ctx *fasthttp.RequestCtx) string {
	return string(ctx.Request.Header.Cookie(refreshCookieName))
}

func getCookieWithRefreshToken(refreshToken string) *fasthttp.Cookie {
	cookie := fasthttp.Cookie{}
	cookie.SetKey(refreshCookieName)
	cookie.SetValue(refreshToken)
	cookie.SetExpire(time.Now().Add(refreshTokenExp))
	cookie.SetHTTPOnly(true)
	cookie.SetSecure(true) // Только если HTTPS
	cookie.SetPath("/")

	return &cookie
}

func getEmptyCookie() *fasthttp.Cookie {
	newCookie := &fasthttp.Cookie{}
	newCookie.SetKey(refreshCookieName)
	newCookie.SetValue("")

	return newCookie
}
