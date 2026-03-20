package main

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/fx"

	"github.com/micoya/gocraft/cfx"
	"github.com/micoya/gocraft/chttp"

	"gocraft-template/app"
)

func main() {
	fx.New(
		fx.WithLogger(cfx.LoggerProvider),

		// 基础设施
		cfx.ProvideConfig[struct{}](),
		cfx.ProvideLogger[struct{}](),
		cfx.ProvideOtel[struct{}](),
		cfx.ProvideDAO[struct{}](),
		cfx.ProvideHTTPServer[struct{}](),

		// 业务层
		app.Module(),

		// 路由挂载（apiserver 独有）
		fx.Invoke(app.RegisterRoutes),

		// 打印启动地址
		fx.Invoke(func(lc fx.Lifecycle, s *chttp.Server) {
			lc.Append(fx.Hook{
				OnStart: func(_ context.Context) error {
					host := strings.Replace(s.Addr(), ":", "localhost:", 1)
					fmt.Printf("\n\033[32m🚀 服务已启动 → http://%s\033[0m\n\n", host)
					return nil
				},
			})
		}),
	).Run()
}
