package app

import (
	"go.uber.org/fx"

	"github.com/micoya/gocraft/cfx"

	"gocraft-template/app/api"
	"gocraft-template/app/service"
)

// InfraModule 提供核心基础设施依赖：配置、日志、OTel、HTTP Server。
func InfraModule() fx.Option {
	return cfx.CoreModule[struct{}]()
}

// ServiceModule 提供所有业务 Service 依赖，包含基础设施。
// 适用于需要调用业务逻辑但不启动 HTTP 服务的 cmd（如消费者、定时任务）。
func ServiceModule() fx.Option {
	return fx.Options(
		InfraModule(),
		fx.Provide(
			service.NewHelloService,
		),
	)
}

// APIModule 提供完整的 HTTP API 服务依赖，包含业务服务、路由注册。
// 适用于 cmd/apiserver。
func APIModule() fx.Option {
	return fx.Options(
		ServiceModule(),
		fx.Provide(
			api.NewHelloHandler,
		),
		fx.Invoke(RegisterRoutes),
	)
}
