package app

import (
	"go.uber.org/fx"

	"gocraft-template/app/api"
	"gocraft-template/app/service"
)

// Module 汇总所有业务依赖（Service、Handler、pkg 单例）。
//
// 在各 cmd 的 fx.New 中引入即可复用全部业务层依赖，
// 路由挂载等传输层细节由各 cmd 自行处理。
func Module() fx.Option {
	return fx.Options(
		fx.Provide(
			service.NewHelloService,
		),
		fx.Provide(
			api.NewHelloHandler,
		),
	)
}
