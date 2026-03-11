package app

import (
	"context"
	"log/slog"

	"go.uber.org/fx"
	"github.com/micoya/gocraft/chttp"
	"github.com/micoya/gocraft/clog"
	"github.com/micoya/gocraft/config"
	"github.com/micoya/gocraft/cotel"

	"gocraft-template/app/api"
	"gocraft-template/app/service"
)

func provideConfig() (*config.Config[struct{}], error) {
	return config.Load[struct{}](context.Background())
}

func provideLogger(cfg *config.Config[struct{}]) (*slog.Logger, error) {
	return clog.NewFromConfig(cfg.Log)
}

func provideOtelProvider(lc fx.Lifecycle, cfg *config.Config[struct{}]) (*cotel.Provider, error) {
	otelCfg := cfg.Otel
	if otelCfg == nil {
		otelCfg = &config.OtelConfig{} // 默认启用，endpoint 为空时仅本地生成 span
	}
	svcName := cfg.Name
	if svcName == "" {
		svcName = "gocraft-template"
	}
	p, err := cotel.New(context.Background(), otelCfg, svcName)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return p.Shutdown(ctx)
		},
	})
	return p, nil
}

func provideServer(cfg *config.Config[struct{}], log *slog.Logger, otel *cotel.Provider) *chttp.Server {
	return chttp.New(
		chttp.WithServerConfig(cfg.HTTPServer),
		chttp.WithLogger(log),
		chttp.WithOtelProvider(otel),
	)
}

// InfraModule 提供基础设施依赖：配置、日志。
// 模板 demo 模式：无数据库，user 等接口返回固定数据。
func InfraModule() fx.Option {
	return fx.Options(
		fx.Provide(
			provideConfig,
			provideLogger,
		),
	)
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

// APIModule 提供完整的 HTTP API 服务依赖，包含业务服务、路由和 HTTP 生命周期。
// 适用于 cmd/apiserver。
func APIModule() fx.Option {
	return fx.Options(
		ServiceModule(),
		fx.Provide(
			provideOtelProvider,
			provideServer,
			api.NewHelloHandler,
		),
		fx.Invoke(RegisterRoutes),
	)
}
