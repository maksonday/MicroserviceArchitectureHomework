package service

import (
	"auth/config"
	"strings"

	"github.com/valyala/fasthttp"
)

const maxBodySize = 16 << 20 // 16 MB

func NewServer(config *config.ServerConfig) *fasthttp.Server {
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			parts := strings.Split(strings.Trim(string(ctx.Path()), "/"), "/")
			switch parts[0] {
			case "health":
				healthCheckHandler(ctx)
			case "refresh":
				refreshHandler(ctx)
			case "login":
				loginHandler(ctx)
			case "logout":
				logoutHandler(ctx)
			case "register":
				registerHandler(ctx)
			default:
				ctx.Error("not found", fasthttp.StatusNotFound)
			}
		},

		MaxRequestBodySize: maxBodySize,
		ReadTimeout:        config.ReadTimeout,
		WriteTimeout:       config.WriteTimeout,
		IdleTimeout:        config.IdleTimeout,
		Concurrency:        config.Concurrency,
	}

	return s
}
