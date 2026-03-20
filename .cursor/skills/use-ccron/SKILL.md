---
name: use-ccron
description: 使用 ccron 定时任务调度器，支持分布式防重复执行
---

## 概述

`ccron` 提供基于 `robfig/cron` 的定时任务调度器，支持 OTel 全链路追踪和分布式防重复执行（需配合 `clocker`）。

```go
import "github.com/micoya/gocraft/ccron"
```

---

## 普通模式（单实例）

```go
scheduler := ccron.New(
    ccron.WithTimezone("Asia/Shanghai"),
    ccron.WithLogger(logger),
)

// 注册任务（6 位 cron：秒 分 时 日 月 周）
scheduler.Add("清理过期订单", "0 0 2 * * *", func(ctx context.Context) {
    // ctx 已注入 OTel span，直接透传给下游调用
    if err := orderSvc.CleanExpired(ctx); err != nil {
        slog.ErrorContext(ctx, "清理失败", "error", err)
    }
})

scheduler.Start()
defer scheduler.Stop(ctx) // 优雅停止，等待当前执行的任务完成
```

---

## 分布式模式（多实例防重复）

多实例部署时，每次任务触发时会竞争分布式锁，只有抢到锁的节点执行，其余节点静默跳过。

```go
import (
    "github.com/micoya/gocraft/ccron"
    "github.com/micoya/gocraft/clocker"
    "github.com/micoya/gocraft/cdao/redisx"
)

locker := clocker.New(redisx.Must(dao))

scheduler := ccron.New(
    ccron.WithTimezone("Asia/Shanghai"),
    ccron.WithLocker(locker, 5*time.Minute), // LockTTL 略大于任务预期最长执行时间
)

scheduler.Add("每日结算", "0 0 1 * * *", func(ctx context.Context) {
    settlementSvc.RunDaily(ctx)
})

scheduler.Start()
```

---

## config.yaml 配置

```yaml
cron:
  timezone: "Asia/Shanghai"
  distributed: true
  lock_redis: "default"  # 对应 dao.redis 中的实例名
  lock_ttl: 5m
```

从 config 读取配置：

```go
var opts []ccron.Option
opts = append(opts, ccron.WithTimezone(cfg.Cron.Timezone))

if cfg.Cron.Distributed {
    locker := clocker.New(redisx.Must(dao, cfg.Cron.LockRedis))
    opts = append(opts, ccron.WithLocker(locker, cfg.Cron.LockTTL))
}

scheduler := ccron.New(opts...)
```

---

## 通过 fx 集成（推荐）

`cfx.ProvideCron` 自动从 `config.yaml` 读取时区、分布式配置，并管理 Start/Stop 生命周期。
分布式模式下还需同时注册 `cfx.ProvideLocker`（它会自动注入到 `ProvideCron`）。

**`app/module.go`** — 注册任务（在 `fx.Invoke` 中）：

```go
// app/module.go
func Module() fx.Option {
    return fx.Options(
        fx.Provide(service.NewOrderService),

        // 注册定时任务（scheduler 和 locker 由 main.go 的 cfx.Provide* 提供）
        fx.Invoke(registerCronJobs),
    )
}

func registerCronJobs(scheduler *ccron.Scheduler, svc *service.OrderService, log *slog.Logger) {
    scheduler.Add("清理过期订单", "0 0 2 * * *", func(ctx context.Context) {
        if err := svc.CleanExpired(ctx); err != nil {
            log.ErrorContext(ctx, "清理失败", slog.Any("error", err))
        }
    })
}
```

**`cmd/apiserver/main.go`** — 注册基础设施：

```go
fx.New(
    cfx.ProvideConfig[AppConfig](),
    cfx.ProvideDAO[AppConfig](),
    cfx.ProvideLocker(),        // 分布式模式需要，无分布式可省略
    cfx.ProvideCron[AppConfig](), // 自动读取 cfg.Cron 配置
    app.Module(),
).Run()
```

**`config.yaml`** — 开启分布式模式：

```yaml
cron:
  timezone: "Asia/Shanghai"
  distributed: true   # true 时 cfx.ProvideCron 自动启用 Locker
  lock_redis: default
  lock_ttl: 5m
```

---

## Cron 表达式参考（6 位含秒）

```
┌─────────── 秒 (0-59)
│ ┌─────────── 分 (0-59)
│ │ ┌─────────── 时 (0-23)
│ │ │ ┌─────────── 日 (1-31)
│ │ │ │ ┌─────────── 月 (1-12)
│ │ │ │ │ ┌─────────── 周 (0-6, 0=周日)
│ │ │ │ │ │
* * * * * *
```

| 表达式 | 含义 |
|---|---|
| `0 0 2 * * *` | 每天 02:00:00 |
| `0 30 9 * * 1-5` | 工作日 09:30:00 |
| `0 */5 * * * *` | 每 5 分钟（整分钟时执行）|
| `0 0 */6 * * *` | 每 6 小时整点执行 |
| `@hourly` | 每小时（等同于 `0 0 * * * *`）|

---

## 动态管理任务

```go
id, err := scheduler.Add("临时任务", "0 * * * * *", fn)

// 稍后移除
scheduler.Remove(id)
```
