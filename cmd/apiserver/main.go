package main

import (
	"go.uber.org/fx"

	"github.com/micoya/gocraft/cfx"

	"gocraft-template/app"
)

func main() {
	fx.New(
		fx.WithLogger(cfx.LoggerProvider),
		app.APIModule(),
		fx.Invoke(cfx.RunHTTPServer),
	).Run()
}
