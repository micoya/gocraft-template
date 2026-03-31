---
name: use-climiter
description: 使用 climiter 限流器（Redis 分布式滑动窗口 / 固定窗口 / 令牌桶 / 本地内存）及配置驱动多实例 Registry
---

## 配置

在 `config.yaml` 中添加 `limiter` 块（顶层，与 `dao` 同级）：

```yaml
limiter:
  # 推荐：分布式滑动窗口（边界平滑，无需锁）
  default:
    algo: sliding_window   # 算法：sliding_window | fixed_window | token_bucket | local
    rate: 100              # 每 window 时间内最大请求数
    window: 1s             # 时间窗口大小
    redis: default         # cdao 中 Redis 实例名，默认 "default"

  # 严格接口限流：令牌桶（支持突发）
  strict:
    algo: token_bucket
    rate: 10
    window: 1m
    burst: 5               # 最大突发容量（仅 token_bucket 生效）
    redis: default

  # 无 Redis 依赖的单机限流（仅单实例或测试环境使用）
  local:
    algo: local
    rate: 50
    window: 1s
```

> `algo` 为 `local` 时无需配置 `redis` 字段。其余算法需保证 `dao.redis` 中对应实例已配置。

---

## cfx / fx 注入（Registry 方式，推荐）

在 `app/module.go` 中注册 `*climiter.Registry`：

```go
import (
    "github.com/micoya/gocraft/cdao"
    "github.com/micoya/gocraft/cdao/redisx"
    "github.com/micoya/gocraft/climiter"
    "github.com/micoya/gocraft/config"
    "go.uber.org/fx"
)

// 在 app.Module() 的 fx.Options 中添加
fx.Provide(func(dao *cdao.DAO, cfg *config.Config[AppConfig]) (*climiter.Registry, error) {
    return climiter.NewFromConfig(cfg.Limiter, func(name string) (*goredis.Client, error) {
        return redisx.Must(dao, name), nil
    })
}),
```

---

## 算法选择

| 算法 | 配置值 | 特点 | 推荐场景 |
|---|---|---|---|
| 分布式滑动窗口 | `sliding_window` | 边界平滑，无锁，性能好（**首选**） | API 限流、用户请求频率控制 |
| 分布式固定窗口 | `fixed_window` | 资源消耗最低，实现最简 | 低流量、资源敏感场景 |
| 分布式令牌桶 | `token_bucket` | 支持突发，需分布式锁 | 外部 API 调用平滑、允许短时集中请求 |
| 单机内存滑动窗口 | `local` | 零 Redis 依赖，进程内独立计数 | 单实例、开发 / 测试环境 |

---

## 使用 Registry（配置驱动，推荐）

```go
// Handler 或 Service 注入 *climiter.Registry
type UserHandler struct {
    limiter *climiter.Registry
    svc     *UserService
}

func NewUserHandler(limiter *climiter.Registry, svc *UserService) *UserHandler {
    return &UserHandler{limiter: limiter, svc: svc}
}

func (h *UserHandler) SendCode(c *gin.Context) {
    userID := c.GetString("user_id")

    retryAfter, err := h.limiter.Must("strict").Limit(c.Request.Context(), "send_code:"+userID)
    if errors.Is(err, climiter.ErrRateLimited) {
        common.Fail(c, CodeRateLimited, fmt.Sprintf("请求过于频繁，请 %.0f 秒后重试", retryAfter.Seconds()))
        return
    }
    if err != nil {
        common.InternalError(c, "限流检查失败")
        return
    }

    // 通过限流，执行业务逻辑
    common.OK(c, nil)
}
```

---

## 直接构造（不使用配置文件）

```go
import (
    "github.com/micoya/gocraft/climiter"
    "github.com/micoya/gocraft/cdao/redisx"
)

// 分布式滑动窗口（推荐）
lim := climiter.NewSlidingWindowRedis(
    redisx.Must(dao),
    100,          // rate: 每窗口最大请求数
    time.Second,  // window: 时间窗口
    climiter.WithKeyPrefix("myapp:api:"),  // 可选：自定义 key 前缀
)

// 分布式固定窗口
lim := climiter.NewFixedWindowRedis(redisx.Must(dao), 100, time.Second)

// 分布式令牌桶（突发容量 10）
lim := climiter.NewTokenBucketRedis(
    redisx.Must(dao),
    10,           // rate: 每 window 补充的令牌数
    time.Minute,  // window
    climiter.WithBurst(10),
)

// 单机内存滑动窗口
lim := climiter.NewLocalSlidingWindow(50, time.Second)
```

---

## 限流检查

```go
retryAfter, err := lim.Limit(ctx, "user:123")
if errors.Is(err, climiter.ErrRateLimited) {
    // 被限流，retryAfter 为建议最短等待时间
    return fmt.Errorf("rate limited, retry after %v", retryAfter)
}
if err != nil {
    return fmt.Errorf("limiter error: %w", err)
}
// 通过，继续执行
```

---

## 在 Middleware 中使用

```go
// 全局 IP 限流中间件
func RateLimitByIP(reg *climiter.Registry) gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()
        retryAfter, err := reg.Must("default").Limit(c.Request.Context(), ip)
        if errors.Is(err, climiter.ErrRateLimited) {
            c.Header("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
            common.Fail(c, CodeRateLimited, "请求过于频繁")
            c.Abort()
            return
        }
        c.Next()
    }
}
```

---

## OTel 链路追踪

`climiter` 内置 OpenTelemetry 支持，每次 `Limit` 调用都会自动创建 span，无需额外配置。

Span 属性：

| 属性 | 说明 |
|---|---|
| `ratelimit.key` | 被检查的 key |
| `ratelimit.algo` | 限流算法 |
| `ratelimit.name` | Registry 实例名（通过 `NewFromConfig` 创建时存在） |
| `ratelimit.allowed` | 是否放行（true/false） |
| `ratelimit.retry_after_ms` | 被限流时的建议等待毫秒数 |

---

## 注意事项

- `key` 建议包含业务维度前缀（如 `"sms:"+userID`），避免不同接口共享计数器
- `NewTokenBucketRedis` 内部使用 Redis 分布式锁（基于 redsync），key 量大时注意锁资源
- `local` 算法多副本部署时各实例独立计数，实际总放量 = `rate × 副本数`，多实例请使用 Redis 算法
- `Registry.Must` 找不到实例名时会 `panic`，建议用 `Get` 做安全检查或确保配置完整
- 在 Handler 中被限流应返回 HTTP 429，可在响应头加 `Retry-After`
