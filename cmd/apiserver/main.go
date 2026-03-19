package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"go.uber.org/fx"

	"github.com/micoya/gocraft/cfx"
	"github.com/micoya/gocraft/chttp"
	"github.com/micoya/gocraft/config"
	"github.com/micoya/gocraft/cotel"

	"gocraft-template/app"
	"gocraft-template/app/api"
	"gocraft-template/app/service"
)

func main() {
	fx.New(
		fx.WithLogger(cfx.LoggerProvider),

		// 基础设施
		cfx.ProvideConfig[struct{}](),
		cfx.ProvideLogger[struct{}](),
		cfx.ProvideOtel[struct{}](),
		cfx.ProvideDAO[struct{}](),

		// 业务服务
		fx.Provide(
			service.NewHelloService,
		),

		// HTTP Server
		fx.Provide(func(lc fx.Lifecycle, cfg *config.Config[struct{}], log *slog.Logger, otel *cotel.Provider) *chttp.Server {
			server := chttp.New(
				chttp.WithServerConfig(cfg.HTTPServer),
				chttp.WithLogger(log),
				chttp.WithOtelProvider(otel),
			)
			ctx, cancel := context.WithCancel(context.Background())
			lc.Append(fx.Hook{
				OnStart: func(_ context.Context) error {
					go func() {
						if err := server.Run(ctx); err != nil {
							log.Error("http server stopped", slog.Any("error", err))
						}
					}()
					addr := server.Addr()
					host := strings.Replace(addr, ":", "localhost:", 1)
					fmt.Printf("\n\033[32m🚀 服务已启动 → http://%s\033[0m\n\n", host)
					return nil
				},
				OnStop: func(_ context.Context) error {
					cancel()
					return nil
				},
			})
			return server
		}),

		// Handler 与路由
		fx.Provide(
			api.NewHelloHandler,
		),
		fx.Invoke(app.RegisterRoutes),
	).Run()
}
