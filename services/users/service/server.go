package service

import (
	"strings"
	"users/config"

	"github.com/valyala/fasthttp"
)

const maxBodySize = 16 << 20 // 16 MB

func NewServer(config *config.ServerConfig) *fasthttp.Server {
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			parts := strings.Split(strings.Trim(string(ctx.Path()), "/"), "/")
			switch parts[0] {
			case "user":
				switch {
				case len(parts) == 1:
					createUser(ctx)
				case len(parts) == 2:
					handleUser(ctx, parts[1])
				default:
					ctx.Error("not found", fasthttp.StatusNotFound)
					return
				}
			case "health":
				healthCheckHandler(ctx)
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
