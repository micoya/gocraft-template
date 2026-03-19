---
name: use-dao
description: 使用Redis, Gorm, MemoryCache, OpenAI, OSS, MongoDB, Elasticsearch, Kafka, RabbitMQ, HTTPClient
---

## 使用方式

1. 确保 `config.yaml` 已包含对应的配置(例如Redis需要`dao.redis`配置), 不知道什么配置可以查看`config`结构体的定义。如果这一步未完成应该通知用户先完成，可以直接告诉用户需要的东西以双方交互的东西完成配置。
2. 引入 `github.com/micoya/gocraft/cdao/<Provider>` 包
3. 确保`main`包中已引用对应的驱动 `github.com/micoya/gocraft/cdao/provider/<Provider>`

## 支持的Dao

### Redis

**使用包：**`github.com/micoya/gocraft/cdao/redisx`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/redis`

**使用方式:** `redisx.Must(*cdao.DAO, name...) *redis.Client`

**config.yaml 配置：**
```yaml
dao:
  redis:
    default:
      addr: "127.0.0.1:6379"
      password: ""
      db: 0
```

**备注**

返回的是`github.com/redis/go-redis/v9` 的 `Client`

---

### Gorm（数据库）

**使用包：**`github.com/micoya/gocraft/cdao/gormx`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/database`

**使用方式:** `gormx.Must(*cdao.DAO, name...) *gorm.DB`

**config.yaml 配置：**
```yaml
dao:
  database:
    default:
      driver: "mysql"  # 或 "postgres"
      dsn: "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True"
```

**备注**

返回的是`gorm.io/gorm` 的 `DB`，支持 `mysql` 和 `postgres` 两种 driver

---

### OSS（阿里云对象存储）

**使用包：**`github.com/micoya/gocraft/cdao/ossx`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/oss`

**使用方式:** `ossx.Must(*cdao.DAO, name...) *oss.Client`

**config.yaml 配置：**
```yaml
dao:
  oss:
    default:
      endpoint: "https://oss-cn-hangzhou.aliyuncs.com"
      access_key_id: ""
      access_key_secret: ""
```

**备注**

返回的是`github.com/aliyun/aliyun-oss-go-sdk/oss` 的 `Client`

---

### OpenAI

**使用包：**`github.com/micoya/gocraft/cdao/openaix`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/openai`

**使用方式:** `openaix.Must(*cdao.DAO, name...) *openai.Client`

**config.yaml 配置：**
```yaml
dao:
  openai:
    default:
      api_key: ""
      base_url: ""  # 留空使用官方地址；可接入 DeepSeek、通义千问等兼容服务
```

**备注**

返回的是`github.com/openai/openai-go` 的 `Client`，支持通过 `base_url` 接入任意 OpenAI 兼容服务（DeepSeek、通义千问等）

---

### MemoryCache（内存缓存）

**使用包：**`github.com/micoya/gocraft/cdao/mcachex`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/mcache`

**使用方式:** `mcachex.Must(*cdao.DAO, name...) *ristretto.Cache[string, any]`

**config.yaml 配置：**
```yaml
dao:
  mcache:
    default:
      num_counters: 10000000  # 预期最大条目数的 10 倍
      max_cost: 536870912     # 最大内存 512MB（字节）
```

**备注**

返回的是`github.com/dgraph-io/ristretto/v2` 的 `Cache[string, any]`，基于 ristretto 高性能并发缓存实现

---

### MongoDB

**使用包：**`github.com/micoya/gocraft/cdao/mongox`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/mongo`

**使用方式:** `mongox.Must(*cdao.DAO, name...) *mongo.Client`

**config.yaml 配置：**
```yaml
dao:
  mongo:
    default:
      uri: "mongodb://127.0.0.1:27017"
      connect_timeout: 10s
```

**备注**

返回的是`go.mongodb.org/mongo-driver/mongo` 的 `Client`

---

### Elasticsearch

**使用包：**`github.com/micoya/gocraft/cdao/elasticsearchx`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/elasticsearch`

**使用方式:** `elasticsearchx.Must(*cdao.DAO, name...) *elasticsearch.Client`

**config.yaml 配置：**
```yaml
dao:
  elasticsearch:
    default:
      addresses:
        - "http://localhost:9200"
      username: ""
      password: ""
      api_key: ""   # 优先级高于 username/password
      cloud_id: ""  # Elastic Cloud 部署 ID
```

**备注**

返回的是`github.com/elastic/go-elasticsearch/v8` 的 `Client`

---

### Kafka

**使用包：**`github.com/micoya/gocraft/cdao/kafkax`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/kafka`

**使用方式:** `kafkax.Must(*cdao.DAO, name...) *kafka.Client`

**config.yaml 配置：**
```yaml
dao:
  kafka:
    default:
      brokers:
        - "kafka:9092"
      dial_timeout: 10s
```

**备注**

返回的是框架封装的 `*kafka.Client`（`github.com/micoya/gocraft/cdao/provider/kafka`），提供带 OTel 追踪的 Writer/Reader：

```go
client := kafkax.Must(dao)

// 创建 Producer（自动注入 trace context）
writer := client.NewWriter("order.created")
defer writer.Close()
err := writer.WriteMessages(ctx, kafkago.Message{
    Key:   []byte("order-123"),
    Value: []byte(`{"order_id":123}`),
})

// 创建 Consumer（自动提取 trace context）
reader := client.NewReader(kafkago.ReaderConfig{
    GroupID: "order-service",
    Topic:   "order.created",
})
defer reader.Close()
msgCtx, msg, err := reader.ReadMessage(ctx)
defer trace.SpanFromContext(msgCtx).End()
// 使用 msgCtx 处理消息，span 在处理完成后结束
```

底层使用 `github.com/segmentio/kafka-go`

---

### RabbitMQ

**使用包：**`github.com/micoya/gocraft/cdao/rabbitmqx`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/rabbitmq`

**使用方式:** `rabbitmqx.Must(*cdao.DAO, name...) *amqp.Connection`

**config.yaml 配置：**
```yaml
dao:
  rabbitmq:
    default:
      url: "amqp://guest:guest@localhost:5672/"
```

**备注**

返回的是`github.com/rabbitmq/amqp091-go` 的 `*amqp.Connection`

---

### HTTPClient（外部 HTTP 服务客户端）

**使用包：**`github.com/micoya/gocraft/cdao/httpclientx`

**需要引用驱动：**`github.com/micoya/gocraft/cdao/provider/httpclient`

**使用方式:** `httpclientx.Must(*cdao.DAO, name...) *http.Client`

**config.yaml 配置：**
```yaml
dao:
  http_client:
    payment:
      base_url: "https://payment.internal"
      timeout: 30s
      max_idle_conns: 100
      max_conns_per_host: 0
      idle_conn_timeout: 90s
      tls_skip_verify: false

      # 重试配置（不填则不重试）
      retry:
        max_attempts: 3   # 最多重试 3 次（不含首次），共 4 次尝试
        wait_min: 100ms
        wait_max: 2s

      # 熔断器配置（不填则不启用）
      circuit_breaker:
        max_requests: 1   # 半开状态探测请求数
        interval: 60s     # 统计窗口
        timeout: 30s      # 熔断持续时间
        threshold: 5      # 连续失败次数触发熔断
```

**备注**

返回标准库 `*net/http.Client`，内置 OTel 追踪、重试（指数退避+抖动）、熔断器，业务代码无感知直接使用。重试策略：幂等方法（GET/HEAD/PUT/DELETE/OPTIONS）5xx 自动重试；POST 等非幂等方法仅在网络错误时重试。
