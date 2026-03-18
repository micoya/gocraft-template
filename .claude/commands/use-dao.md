---
name: use-dao
description: 使用Redis, Gorm, MemoryCache, OpenAI, OSS
---

## 使用方式

1. 确保 `config.yaml` 已包含对应的配置(例如Redis需要`dao.redis`配置), 不知道什么配置可以查看`config`结构体的定义。如果这一步未完成应该通知用户先完成，可以直接告诉用户需要的东西以双方交互的东西完成配置。
2. 引入 `github.com/micoya/gocraft/cdao/<Provider>` 包
3. 确保`main`包中已引用对应的驱动 `github.com/micoya/gocraft/cdao/provider/<Provider>`

## 支持的Dao

### Redis

**使用包：**`github.com/micoya/gocraft/cdao/redisx`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/redis`

**使用方式:** `redisx.Must(*cdao.Dao, name...) *redis.Client`

**备注**

返回的是`github.com/redis/go-redis/v9` 的 `Client`

---

### Gorm（数据库）

**使用包：**`github.com/micoya/gocraft/cdao/gormx`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/database`

**使用方式:** `gormx.Must(*cdao.Dao, name...) *gorm.DB`

**备注**

返回的是`gorm.io/gorm` 的 `DB`，支持 `mysql` 和 `postgres` 两种 driver

---

### OSS（阿里云对象存储）

**使用包：**`github.com/micoya/gocraft/cdao/ossx`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/oss`

**使用方式:** `ossx.Must(*cdao.Dao, name...) *oss.Client`

**备注**

返回的是`github.com/aliyun/aliyun-oss-go-sdk/oss` 的 `Client`

---

### OpenAI

**使用包：**`github.com/micoya/gocraft/cdao/openaix`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/openai`

**使用方式:** `openaix.Must(*cdao.Dao, name...) *openai.Client`

**备注**

返回的是`github.com/openai/openai-go` 的 `Client`，支持通过 `base_url` 接入任意 OpenAI 兼容服务（DeepSeek、通义千问等）

---

### MemoryCache（内存缓存）

**使用包：**`github.com/micoya/gocraft/cdao/mcachex`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/mcache`

**使用方式:** `mcachex.Must(*cdao.Dao, name...) *ristretto.Cache[string, any]`

**备注**

返回的是`github.com/dgraph-io/ristretto/v2` 的 `Cache[string, any]`，基于 ristretto 高性能并发缓存实现
