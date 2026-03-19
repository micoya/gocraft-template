---
name: use-clocker
description: 使用 clocker 分布式锁，基于 Redis 实现互斥执行
---

## 概述

`clocker` 提供基于 Redis 的分布式锁，适用于多实例部署时需要互斥执行的场景（库存扣减、分布式幂等、防重复任务等）。

| 机制 | 实现 |
|---|---|
| 加锁 | `SET key token NX PX ttl`（原子操作）|
| 解锁 | Lua 脚本：验证 token 后才删除，防止误删他人锁 |
| 阻塞等待 | 指数退避重试 |
| 自动续期 | Watchdog goroutine 每 TTL/3 刷新过期时间 |

---

## 前置条件

确保 `config.yaml` 中已配置 Redis（`dao.redis`），并在 `main` 包中引入 Redis DAO 驱动。

---

## 创建锁实例

```go
import (
    "github.com/micoya/gocraft/clocker"
    "github.com/micoya/gocraft/cdao/redisx"
)

locker := clocker.New(redisx.Must(dao))

// 使用专用 Redis 实例（推荐与业务数据隔离）
locker := clocker.New(redisx.Must(dao, "lock"))

// 自定义配置
locker := clocker.New(
    redisx.Must(dao),
    clocker.WithKeyPrefix("myapp:lock:"),
    clocker.WithRetryInterval(50*time.Millisecond, 1*time.Second),
)
```

---

## TryLock（非阻塞，推荐用于幂等处理）

```go
lock, err := locker.TryLock(ctx, "order:pay:123", 30*time.Second)
if errors.Is(err, clocker.ErrLockNotAcquired) {
    // 锁已被其他实例持有，直接返回（幂等处理）
    return nil
}
if err != nil {
    return fmt.Errorf("获取锁失败: %w", err)
}
defer lock.Unlock(ctx) // 确保释放

// 执行临界区逻辑
return processPayment(ctx, orderID)
```

---

## Lock（阻塞等待，适合串行化操作）

```go
// 最多等待 10 秒
lockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()

lock, err := locker.Lock(lockCtx, "inventory:sku:456", 30*time.Second)
if err != nil {
    return err // 超时或 ctx 取消
}
defer lock.Unlock(ctx)

return deductInventory(ctx, skuID, quantity)
```

---

## config.yaml 配置（专用 Redis 实例）

```yaml
dao:
  redis:
    default:
      addr: "127.0.0.1:6379"
    lock:
      addr: "127.0.0.1:6379"
      db: 1
```

---

## 注意事项

- `ttl` 应略大于业务执行预期最长时间（Watchdog 会自动续期）
- `Unlock` 是幂等的，锁已过期或不存在时静默返回
- 与 `ccron` 配合可实现分布式定时任务防重复执行（参考 use-ccron skill）
- 单 Redis 节点存在单点故障风险；生产高可用场景建议使用 Redis Cluster
