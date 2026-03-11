package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"github.com/micoya/gocraft/chttp"
	"github.com/micoya/gocraft/config"

	"gocraft-template/app"
)

func runHTTPServer(lc fx.Lifecycle, server *chttp.Server, cfg *config.Config[struct{}], log *slog.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := server.Run(ctx); err != nil {
					log.Error("http server exited with error", slog.Any("error", err))
				}
			}()
			addr := cfg.HTTPServer.Addr
			host := strings.Replace(addr, ":", "localhost:", 1)
			fmt.Printf("\n\033[32m🚀 服务已启动 → http://%s\033[0m\n\n", host)
			return nil
		},
		OnStop: func(_ context.Context) error {
			cancel()
			return nil
		},
	})
}

func main() {
	fx.New(
		fx.WithLogger(func(log *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: log}
		}),
		app.APIModule(),
		fx.Invoke(runHTTPServer),
	).Run()
}
