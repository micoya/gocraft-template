---
name: use-ccache
description: 使用 ccache 多级缓存（Redis缓存、内存缓存、两级联合缓存）
---

## 概述

`ccache` 提供统一的缓存抽象接口与三种实现：Redis、内存（ristretto）和两级联合缓存（L1 内存 + L2 Redis）。所有实现缓存值类型为 `string`，业务层自行 JSON 序列化/反序列化。

| 实现 | 特点 | 适用场景 |
|---|---|---|
| `RedisCache` | 持久化、跨进程共享、TTL 精确 | 分布式缓存、会话、共享热点数据 |
| `MemoryCache` | 零延迟、进程内 | 高频读取、本地计算结果缓存 |
| `LayeredCache` | L1 内存 miss 时查 L2 Redis 并回填 | 两全其美，降低 Redis 压力（**推荐生产使用**）|

---

## 前置条件

- 使用 `RedisCache` / `LayeredCache` 需确保 `config.yaml` 中已配置 Redis（`dao.redis`），并在 `main` 包中引入 Redis DAO 驱动。

---

## 创建缓存实例

### 单级 Redis 缓存

```go
import (
    "github.com/micoya/gocraft/ccache"
    "github.com/micoya/gocraft/cdao/redisx"
)

cache := ccache.NewRedis(
    redisx.Must(dao),
    ccache.WithRedisDefaultTTL(time.Hour),
    ccache.WithRedisKeyPrefix("myapp:"),
)
```

### 单级内存缓存

```go
cache, err := ccache.NewMemoryFromConfig(
    10_000_000,  // numCounters（预期条目数的 10 倍）
    512<<20,     // maxCost（512MB）
    ccache.WithMemoryDefaultTTL(5*time.Minute),
)
```

### 两级缓存（推荐生产环境）

```go
l1, _ := ccache.NewMemoryFromConfig(10_000_000, 512<<20,
    ccache.WithMemoryDefaultTTL(5*time.Minute),
)
l2 := ccache.NewRedis(redisx.Must(dao),
    ccache.WithRedisDefaultTTL(time.Hour),
)
cache := ccache.NewLayered(l1, l2,
    ccache.WithL1TTL(5*time.Minute), // L1 回填时使用较短 TTL，避免内存数据过旧
)
```

---

## 接口说明

```go
// 读取（found=false 表示 key 不存在）
val, found, err := cache.Get(ctx, "user:1")

// 写入（ttl 可选，不传使用默认值）
cache.Set(ctx, "user:1", `{"name":"alice"}`, ccache.TTL(time.Hour))

// 删除（支持多个 key）
cache.Del(ctx, "user:1", "user:2")

// 读取或生成（最常用，防缓存穿透）
val, err := cache.GetOrSet(ctx, "user:1",
    func(ctx context.Context) (string, error) {
        user, err := userRepo.Find(ctx, 1)
        if err != nil {
            return "", err
        }
        b, _ := json.Marshal(user)
        return string(b), nil
    },
    ccache.TTL(time.Hour),
)
```

---

## 在 Service 中使用

```go
type UserService struct {
    cache ccache.Cache
    db    *gorm.DB
    log   *slog.Logger
}

func NewUserService(dao *cdao.DAO, log *slog.Logger) *UserService {
    l1, _ := ccache.NewMemoryFromConfig(10_000_000, 512<<20,
        ccache.WithMemoryDefaultTTL(5*time.Minute),
    )
    l2 := ccache.NewRedis(redisx.Must(dao),
        ccache.WithRedisDefaultTTL(time.Hour),
    )
    return &UserService{
        cache: ccache.NewLayered(l1, l2),
        db:    gormx.Must(dao),
        log:   log,
    }
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*model.User, error) {
    key := fmt.Sprintf("user:%d", id)

    val, err := s.cache.GetOrSet(ctx, key, func(ctx context.Context) (string, error) {
        var user model.User
        if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
            return "", err
        }
        b, _ := json.Marshal(user)
        return string(b), nil
    }, ccache.TTL(time.Hour))

    if err != nil {
        return nil, err
    }

    var user model.User
    _ = json.Unmarshal([]byte(val), &user)
    return &user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, user *model.User) error {
    if err := s.db.WithContext(ctx).Save(user).Error; err != nil {
        return err
    }
    // 更新后删除缓存，下次读取时重新加载
    return s.cache.Del(ctx, fmt.Sprintf("user:%d", user.ID))
}
```
