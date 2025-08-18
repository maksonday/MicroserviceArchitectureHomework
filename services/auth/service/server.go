package service

import (
	"auth/config"
	"strings"

	"github.com/valyala/fasthttp"
)

const maxBodySize = 16 << 20 // 16 MB

func NewServer(config *config.Config) *fasthttp.Server {
	basePath := strings.Trim(config.BasePath, "/")
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			parts := strings.Split(strings.Trim(string(ctx.Path()), "/"), "/")
			if len(parts) < 2 || parts[0] != basePath {
				ctx.Error("not found", fasthttp.StatusNotFound)
				return
			}

			switch parts[1] {
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
		ReadTimeout:        config.ServerConfig.ReadTimeout,
		WriteTimeout:       config.ServerConfig.WriteTimeout,
		IdleTimeout:        config.ServerConfig.IdleTimeout,
		Concurrency:        config.ServerConfig.Concurrency,
	}

	return s
}
