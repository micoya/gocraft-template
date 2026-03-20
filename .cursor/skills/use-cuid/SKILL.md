---
name: use-cuid
description: 使用 cuid 分布式 ID（UUID、Snowflake、Redis 租约 Snowflake、Sonyflake）及 cfx 注入
---

## 配置

| 用法 | 配置块 |
|------|--------|
| 仅代码里写死节点号 / 无配置文件 | 不需要 `uid` 块 |
| `cfx.ProvideUIDSnowflakeStaticFromConfig` / `ProvideUIDRedisSnowflakeFromConfig` | 需要顶层 `uid` 块，见 `github.com/micoya/gocraft/config` 中 `UIDConfig` 与 `config.example.yaml` 注释 |
| Redis 租约 Snowflake | 同时需要 `dao.redis`（与 `uid.redis_snowflake.redis` 实例名一致） |

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

---

## cfx / fx 注入

在 `cmd/*/main.go` 按需增加其一（可与 `ProvideDAO` 组合）：

| 场景 | cfx |
|------|-----|
| UUID v4 | `cfx.ProvideUIDUUID()` → 注入 `cfx.UIDUUIDGen` |
| 固定节点 Snowflake | `cfx.ProvideUIDSnowflakeStatic(nodeID)` 或 `ProvideUIDSnowflakeStaticFromConfig[AppConfig]()` |
| Redis 租约 Snowflake | 先 `cfx.ProvideDAO[AppConfig]()`，再 `ProvideUIDRedisSnowflake("default", opts...)` 或 `ProvideUIDRedisSnowflakeFromConfig[AppConfig]()` |
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

// Sonyflake
id, err := sf.NextID()
```

具体 API 以 `github.com/micoya/gocraft/cuid` 与 `cfx/cuid.go` 为准。
