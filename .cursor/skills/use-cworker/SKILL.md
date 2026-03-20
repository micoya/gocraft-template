---
name: use-cworker
description: 使用 cworker 并发受控的后台任务 Worker 池，避免无限制创建 goroutine
---

## 概述

`cworker` 提供并发受控的后台任务池，解决 Go 中无限制创建 goroutine 导致资源耗尽的问题。

- 信号量限制最大并发 goroutine 数（默认 CPU 核数）
- 每个 goroutine 自动 recover panic，记录堆栈到 slog，不崩溃进程
- 支持优雅关闭：`Stop()` 等待所有在途任务完成后返回
- `TryGo()` 非阻塞提交，立即返回是否成功入队

```go
import "github.com/micoya/gocraft/cworker"
```

---

## 创建 Worker 池

```go
pool := cworker.New(
    cworker.WithConcurrency(50),   // 最大并发数，默认 CPU 核数
    cworker.WithLogger(logger),    // panic 日志记录器，默认 slog.Default()
)
```

---

## 提交任务

```go
// Go：阻塞直到有空闲槽位（推荐用于需要保证执行的任务）
err := pool.Go(ctx, func(ctx context.Context) {
    if err := sendEmail(ctx, email); err != nil {
        slog.ErrorContext(ctx, "发送邮件失败", "error", err)
    }
})
if errors.Is(err, context.Canceled) {
    // pool 已关闭或 ctx 已取消
}

// TryGo：非阻塞提交，达到并发上限时立即返回 false
if !pool.TryGo(ctx, func(ctx context.Context) {
    doWork(ctx)
}) {
    // 达到并发上限，可选择降级处理
    log.Warn("worker pool full, task dropped")
}
```

---

## 在 Handler 中处理异步副作用

```go
func (h *OrderHandler) CreateOrder(c *gin.Context) {
    ctx := c.Request.Context()
    order, err := h.svc.CreateOrder(ctx, req)
    if err != nil {
        common.InternalError(c, err.Error())
        return
    }

    // 异步发送确认邮件，不阻塞响应
    _ = h.pool.Go(c.Request.Context(), func(ctx context.Context) {
        h.notifySvc.SendOrderConfirm(ctx, order)
    })

    common.OK(c, order)
}
```

---

## 通过 fx lifecycle 集成（推荐）

在 `app/module.go` 的 `Module()` 中注册 Worker Pool，fx OnStop 自动优雅关闭：

```go
// app/module.go
func provideWorkerPool(lc fx.Lifecycle, log *slog.Logger) *cworker.Pool {
    pool := cworker.New(
        cworker.WithConcurrency(100),
        cworker.WithLogger(log),
    )
    lc.Append(fx.Hook{
        OnStop: func(ctx context.Context) error {
            return pool.Stop(ctx)
        },
    })
    return pool
}

func Module() fx.Option {
    return fx.Options(
        fx.Provide(
            provideWorkerPool,
            service.NewOrderService,
            api.NewOrderHandler,
        ),
    )
}
```

Worker Pool 需要注入到使用它的 Service 或 Handler：

```go
type OrderHandler struct {
    svc  *service.OrderService
    pool *cworker.Pool
    log  *slog.Logger
}

func NewOrderHandler(svc *service.OrderService, pool *cworker.Pool, log *slog.Logger) *OrderHandler {
    return &OrderHandler{svc: svc, pool: pool, log: log}
}
```
