---
name: use-cuid
description: 使用 cuid 分布式 ID（UUID、Snowflake、Redis 租约 Snowflake、Redis Shortflake 短雪花、Sonyflake）及 cfx 注入
---

## 配置

| 用法 | 配置块 |
|------|--------|
| 仅代码里写死节点号 / 无配置文件 | 不需要 `uid` 块 |
| `cfx.ProvideUIDSnowflakeStaticFromConfig` / `ProvideUIDRedisSnowflakeFromConfig` | 需要顶层 `uid` 块，见 `github.com/micoya/gocraft/config` 中 `UIDConfig` 与 `config.example.yaml` 注释 |
| Redis 租约 Snowflake | 同时需要 `dao.redis`（与 `uid.redis_snowflake.redis` 实例名一致） |
| Redis Shortflake 短雪花 | 需要 `dao.redis`；不需要也不存在 `uid.shortflake` 配置块 |

示例（静态 Snowflake）：

```yaml
uid:
  snowflake_static:
    node_id: 1
```

示例（Redis 租约 Snowflake，需 `dao.redis` 中同名实例）：

```yaml
dao:
  redis:
    default:
      addr: "127.0.0.1:6379"

uid:
  redis_snowflake:
    redis: default
    # key_prefix / heartbeat_every / lease_ttl / max_node_exclusive 可选，见 config.example.yaml
```

示例（Redis Shortflake，需 `dao.redis` 中同名实例；参数在代码中传）：

```yaml
dao:
  redis:
    default:
      addr: "127.0.0.1:6379"
```

---

## cfx / fx 注入

在 `cmd/*/main.go` 按需增加其一（可与 `ProvideDAO` 组合）：

| 场景 | cfx |
|------|-----|
| UUID v4 | `cfx.ProvideUIDUUID()` → 注入 `cfx.UIDUUIDGen` |
| 固定节点 Snowflake | `cfx.ProvideUIDSnowflakeStatic(nodeID)` 或 `ProvideUIDSnowflakeStaticFromConfig[AppConfig]()` |
| Redis 租约 Snowflake | 先 `cfx.ProvideDAO[AppConfig]()`，再 `ProvideUIDRedisSnowflake("default", opts...)` 或 `ProvideUIDRedisSnowflakeFromConfig[AppConfig]()` |
| Redis Shortflake 短雪花 | 先 `cfx.ProvideDAO[AppConfig]()`，再 `ProvideUIDShortflake("default", opts...)` → 注入 `*cuid.Shortflake` |
| Sonyflake | `cfx.ProvideUIDSonyflake()` → `*sonyflake.Sonyflake` |

业务构造函数在 `app/module.go` 中用 `fx.Provide` 注册即可；**无**单独 `cuid` 的 fx 模块，均由上述 `cfx.ProvideUID*` 提供具体类型。

---

## 基本用法

```go
// UUID（实现为 cfx.UIDUUIDGen）
id := gen.NewV4String()

// 静态 Snowflake 节点
id, _ := node.Generate().Int64()

// Redis Snowflake（*cuid.RedisSnowflake）
id, _ := rs.NextID(ctx)

// Redis Shortflake（*cuid.Shortflake）
id, _ := shortflake.Generate(ctx, "orders")
ids, _ := shortflake.GenerateBatch(ctx, "orders", 100)

// Sonyflake
id, err := sf.NextID()
```

## Shortflake 使用边界

`Shortflake` 是 Redis 秒级序列短 ID，不是节点租约型 Snowflake。

- ID 结构：`(当前秒 - epoch) << 20 | sequence`，默认 epoch 为 `2024-01-01 00:00:00 UTC`。
- 可用范围：默认 epoch 起约 136 年；最大 ID 为 `2^52 - 1`，在 JavaScript `Number` 安全整数范围内。
- 每个 `businessKey` 每秒最多生成 `1,048,575` 个序列；超过后等待下一秒。
- 唯一性边界是 `Redis + businessKey`。不同 `businessKey` 可能生成相同数字 ID；全局唯一时所有调用方使用同一个 `businessKey`。
- Redis key 默认为 `snowflake:sequence:<businessKey>:<unixSecond>`，可用 `cuid.WithShortflakeKeyPrefix` 调整前缀。

### Shortflake fx 示例

```go
fx.New(
    cfx.ProvideConfig[AppConfig](),
    cfx.ProvideDAO[AppConfig](),
    cfx.ProvideUIDShortflake("default",
        cuid.WithShortflakeKeyPrefix("app:shortflake:"),
    ),
    app.Module(),
).Run()
```

Service 中直接注入：

```go
type OrderService struct {
    ids *cuid.Shortflake
}

func NewOrderService(ids *cuid.Shortflake) *OrderService {
    return &OrderService{ids: ids}
}

func (s *OrderService) CreateOrder(ctx context.Context) (int64, error) {
    return s.ids.Generate(ctx, "orders")
}
```

具体 API 以 `github.com/micoya/gocraft/cuid` 与 `cfx/cuid.go` 为准。
