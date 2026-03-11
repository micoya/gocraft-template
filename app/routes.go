package app

import (
	"github.com/micoya/gocraft/chttp"

	"gocraft-template/app/api"
)

// RegisterRoutes 注册所有业务路由，依赖由 fx 自动注入。
func RegisterRoutes(
	server *chttp.Server,
	helloHandler *api.HelloHandler,
) {
	server.Engine().GET("/", helloHandler.Welcome)

	v1 := server.Engine().Group("/api/v1")
	{
		v1.GET("/hello", helloHandler.SayHello)
	}
}
