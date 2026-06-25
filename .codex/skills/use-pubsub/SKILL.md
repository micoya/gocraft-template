---
name: use-pubsub
description: 使用 cpubsub（Redis Stream / Kafka / RabbitMQ 发布订阅抽象）
---

## 配置

| 实现 | YAML |
|------|------|
| Redis Stream（`cpubsub/provider/redis`） | **`dao.redis`** |
| Kafka（`cpubsub/provider/kafka`） | **`dao.kafka`** |
| RabbitMQ（`cpubsub/provider/rabbitmq`） | **`dao.rabbitmq`** |

详见 `config.example.yaml` 与 **use-dao** skill。

---

## cfx / fx 注入

1. **`cmd/*/main.go`**：`cfx.ProvideDAO[AppConfig]()`。
2. **`main` 包** blank import 对应驱动：`_ "github.com/micoya/gocraft/cdao/provider/redis"`（或 `kafka` / `rabbitmq`）。
3. **无** `cfx.ProvidePubSub`；在 **`app/module.go`** 用 `fx.Provide` 工厂组装 `cpubsub.PubSub`，或在 Service 构造函数内 `provider/redis.New(redisx.Must(dao))`。

---

## 基本用法（概念）

### Redis Stream 实现

`cpubsub` 的 Redis 实现基于 **Redis Stream 消费组**，底层使用 `XADD` / `XREADGROUP` / `XACK`，支持持久化、消费组、重投递与 ACK。

另有 **Kafka、RabbitMQ** 实现，见 `github.com/micoya/gocraft/cpubsub/provider/kafka`、`.../rabbitmq`，配置与客户端取自对应 `dao` 块。

---

## 前置条件（Redis 版）

1. 已配置 **`dao.redis`**。
2. `main` 中已 **`_` import** `github.com/micoya/gocraft/cdao/provider/redis`。

---

## 核心类型

```go
// 消息结构
type Message struct {
    ID    string // Stream 消息 ID（由 Redis 生成）
    Topic string // 主题名称
    Body  string // 消息内容（字符串，建议 JSON）
}

// 消息处理函数
// 返回 error 时：订阅循环终止，该消息不会被 ACK（后续会重新投递）
type Handler func(ctx context.Context, msg Message) error
```

---

## 创建 PubSub 实例

```go
import (
    "github.com/micoya/gocraft/cdao"
    "github.com/micoya/gocraft/cdao/redisx"
    cpubsubRedis "github.com/micoya/gocraft/cpubsub/provider/redis"
    "github.com/micoya/gocraft/cpubsub"
)

// 基础创建（使用默认配置）
func NewPubSub(dao *cdao.DAO) cpubsub.PubSub {
    client := redisx.Must(dao) // 使用默认 Redis 实例
    return cpubsubRedis.New(client)
}

// 携带选项创建
func NewPubSub(dao *cdao.DAO) cpubsub.PubSub {
    client := redisx.Must(dao)
    return cpubsubRedis.New(client,
        cpubsubRedis.WithPrefix("myapp:"),    // 自定义 Stream Key 前缀，默认 "channel:"
        cpubsubRedis.WithTTL(3*24*time.Hour), // 自定义 TTL，默认 7 天；0 表示永不过期
        cpubsubRedis.WithCompress(true),      // 启用 deflate 压缩，适合消息体较大的场景
    )
}
```

**可用选项说明：**

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithPrefix(string)` | `"channel:"` | Redis Stream Key 前缀，最终 key 为 `{prefix}{topic}` |
| `WithTTL(time.Duration)` | `7 * 24h` | Stream Key 过期时间，`0` 表示永不过期 |
| `WithCompress(bool)` | `false` | 启用 deflate 压缩，消息体 > 1KB 时建议开启 |

---

## 发布消息

```go
// Publish 向指定 topic 发送消息，返回由 Redis 生成的消息 ID
msgID, err := ps.Publish(ctx, "order.created", `{"order_id": 123, "amount": 99.9}`)
if err != nil {
    return fmt.Errorf("publish: %w", err)
}
```

---

## 订阅消息

`Subscribe` 是**阻塞调用**，应在独立 goroutine 中运行（通常通过 fx 生命周期 Hook 启动）。

- **先处理 pending 消息**（异常重启后自动续消费），再切换到新消息
- handler 返回 `nil` → 消息被 ACK，不再重投
- handler 返回 `error` → 消息**不** ACK，订阅循环终止（调用方可决定是否重启）
- `ctx` 取消 → `Subscribe` 返回 `ctx.Err()`

```go
err := ps.Subscribe(ctx, "order.created", "order-service", "consumer-1",
    func(ctx context.Context, msg cpubsub.Message) error {
        // 反序列化消息体
        var payload OrderCreatedPayload
        if err := json.Unmarshal([]byte(msg.Body), &payload); err != nil {
            // 返回 nil 跳过无法解析的消息（避免死循环）
            slog.WarnContext(ctx, "skip invalid message", slog.String("id", msg.ID))
            return nil
        }
        // 处理业务逻辑
        return handleOrderCreated(ctx, payload)
    },
)
```

**参数说明：**

| 参数 | 说明 |
|------|------|
| `topic` | 主题名称，对应 Redis Stream Key 的后缀部分 |
| `group` | 消费组名称，同一 group 的多个 consumer 竞争消费（at-least-once） |
| `consumer` | 消费者名称，同一 group 内唯一标识一个消费者 |
| `handler` | 消息处理函数 |

---

## 在 gocraft-template 中的完整集成

### 1. 在 Service 中注入 PubSub

```go
// app/service/order.go
package service

import (
    "context"
    "encoding/json"
    "log/slog"

    "go.opentelemetry.io/otel"

    "github.com/micoya/gocraft/cdao"
    "github.com/micoya/gocraft/cdao/redisx"
    "github.com/micoya/gocraft/cpubsub"
    cpubsubRedis "github.com/micoya/gocraft/cpubsub/provider/redis"
)

var orderServiceTracer = otel.Tracer("order-service")

type OrderService struct {
    ps  cpubsub.PubSub
    log *slog.Logger
}

func NewOrderService(dao *cdao.DAO, log *slog.Logger) *OrderService {
    client := redisx.Must(dao)
    ps := cpubsubRedis.New(client, cpubsubRedis.WithPrefix("myapp:"))
    return &OrderService{ps: ps, log: log}
}

type OrderCreatedEvent struct {
    OrderID int64   `json:"order_id"`
    Amount  float64 `json:"amount"`
}

// PublishOrderCreated 发布订单创建事件
func (s *OrderService) PublishOrderCreated(ctx context.Context, orderID int64, amount float64) error {
    ctx, span := orderServiceTracer.Start(ctx, "PublishOrderCreated")
    defer span.End()

    body, err := json.Marshal(OrderCreatedEvent{OrderID: orderID, Amount: amount})
    if err != nil {
        return fmt.Errorf("marshal event: %w", err)
    }
    _, err = s.ps.Publish(ctx, "order.created", string(body))
    return err
}

// ConsumeOrderCreated 订阅订单创建事件（阻塞，通过 fx lifecycle 启动）
func (s *OrderService) ConsumeOrderCreated(ctx context.Context) error {
    return s.ps.Subscribe(ctx, "order.created", "order-service", "consumer-1",
        func(ctx context.Context, msg cpubsub.Message) error {
            var event OrderCreatedEvent
            if err := json.Unmarshal([]byte(msg.Body), &event); err != nil {
                s.log.WarnContext(ctx, "skip invalid message", slog.String("id", msg.ID))
                return nil
            }
            s.log.InfoContext(ctx, "received order created event",
                slog.Int64("order_id", event.OrderID),
                slog.Float64("amount", event.Amount),
            )
            // TODO: 处理业务逻辑
            return nil
        },
    )
}
```

### 2. 通过 fx lifecycle 启动消费者

在 `app/module.go` 的 `Module()` 中注册消费者生命周期：

```go
// app/module.go
func Module() fx.Option {
    return fx.Options(
        fx.Provide(
            service.NewOrderService,
        ),
        fx.Invoke(registerOrderConsumer),
    )
}

func registerOrderConsumer(lc fx.Lifecycle, svc *service.OrderService, log *slog.Logger) {
    lc.Append(fx.Hook{
        OnStart: func(_ context.Context) error {
            go func() {
                if err := svc.ConsumeOrderCreated(context.Background()); err != nil {
                    log.Error("order consumer stopped", slog.Any("error", err))
                }
            }()
            return nil
        },
    })
}
```

---

## 注意事项

- **消费语义**：Redis Stream 消费组是 **at-least-once**，handler 需保证**幂等性**
- **消费者命名**：多副本部署时，每个实例应使用不同的 `consumer` 名（例如拼接 Pod IP 或主机名），同一 group 内的多个 consumer 会竞争消费
- **错误处理**：handler 返回 error 会终止订阅循环，适合严重错误；对于可跳过的错误（如消息格式异常），记录日志后返回 `nil`
- **Close**：`cpubsub.PubSub` 实现的 `Close` 不会关闭外部传入的 `*redis.Client`，Redis 连接生命周期由 `cdao` 统一管理
