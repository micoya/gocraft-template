---
name: use-fx-provide
description: 为新建包（pkg/ 或 app/ 下的自定义包）添加 fx 依赖注入，包括如何编写 Provide 函数、在 app/module.go 中统一注册、以及 main.go 中只需调用 app.Module()。当用户提到"依赖注入"、"fx.Provide"、"注入新包"、"pkg 如何注入"时使用。
---

# 为新包添加依赖注入

## 核心约定

- **`app/module.go`** — 所有业务层依赖的注册入口（Service、Handler、pkg 单例）
- **`cmd/*/main.go`** — 只负责基础设施 + `app.Module()` + 各 cmd 独有的 Invoke（如路由挂载）

这样多个 cmd（apiserver / worker / migrator）可以复用同一套业务层，无需重复注册。

---

## 步骤一：在包内实现构造函数

fx 的工作方式：**构造函数参数类型已在容器中 → 自动注入**，无需任何注解。

```go
// pkg/smsclient/smsclient.go
package smsclient

import "log/slog"

type Client struct{ log *slog.Logger }

// New 的参数 *slog.Logger 已在容器中，fx 会自动传入
func New(log *slog.Logger) *Client {
    return &Client{log: log}
}
```

---

## 步骤二：在 app/module.go 中注册

所有业务层的 `fx.Provide` 都集中到这里：

```go
// app/module.go
package app

import (
    "go.uber.org/fx"

    "gocraft-template/app/api"
    "gocraft-template/app/service"
    "gocraft-template/pkg/smsclient"
)

func Module() fx.Option {
    return fx.Options(
        // pkg 单例
        fx.Provide(
            smsclient.New,
        ),

        // Service 层
        fx.Provide(
            service.NewHelloService,
            service.NewOrderService,
        ),

        // Handler 层
        fx.Provide(
            api.NewHelloHandler,
            api.NewOrderHandler,
        ),
    )
}
```

---

## 步骤三：main.go 只调用 app.Module()

```go
// cmd/apiserver/main.go
fx.New(
    cfx.ProvideConfig[AppConfig](),
    cfx.ProvideLogger[AppConfig](),
    cfx.ProvideOtel[AppConfig](),
    cfx.ProvideDAO[AppConfig](),
    cfx.ProvideHTTPServer[AppConfig](),

    app.Module(), // 一行引入全部业务依赖

    // 路由挂载是 apiserver 独有的，不放进 Module
    fx.Invoke(app.RegisterRoutes),
).Run()
```

`cmd/worker/main.go` 复用同一个 `app.Module()`，只是最后 Invoke 不同：

```go
fx.New(
    cfx.ProvideConfig[AppConfig](),
    // ...
    app.Module(),

    fx.Invoke(func(svc *service.OrderService) { /* 启动 worker */ }),
).Run()
```

---

## 特殊情况：需要从业务配置提取参数

当包的构造函数需要 `cfg.App.XXX` 时，在 `app/module.go` 中写一个工厂函数：

```go
// app/module.go
func providePayClient(cfg *config.Config[AppConfig]) *payclient.Client {
    return payclient.New(cfg.App.PaySecret, cfg.App.PayEndpoint)
}

func Module() fx.Option {
    return fx.Options(
        fx.Provide(providePayClient),
        // ...
    )
}
```

---

## 特殊情况：包需要生命周期管理

```go
// app/module.go
func provideWorkerPool(lc fx.Lifecycle, log *slog.Logger) *workerpool.Pool {
    p := workerpool.New(log)
    lc.Append(fx.Hook{
        OnStart: func(_ context.Context) error { p.Start(); return nil },
        OnStop:  func(ctx context.Context) error { return p.Stop(ctx) },
    })
    return p
}
```

---

## 判断速查

| 情况 | 做法 |
|------|------|
| 参数都是基础设施类型 | `fx.Provide(pkg.New)` 加入 `Module()` |
| 需要读取 `cfg.App.XXX` | `app/module.go` 写工厂函数，加入 `Module()` |
| 需要 OnStart/OnStop | `app/module.go` 写工厂函数，注入 `fx.Lifecycle` |
| 路由、定时任务注册等 cmd 特有逻辑 | 留在各 `cmd/*/main.go` 的 `fx.Invoke` |
