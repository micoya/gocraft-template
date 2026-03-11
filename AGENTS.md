# AGENTS.md

本文件描述基于 gocraft 框架的项目开发规范，供 AI 编码助手遵守。

框架README见：https://raw.githubusercontent.com/micoya/gocraft/refs/heads/master/README.md

## 技术栈

- HTTP 框架：`gin`
- 依赖注入：`uber/fx`
- ORM：`gorm`
- 基础设施：`gocraft`（config / clog / cdao / chttp）

## 如何使用Dao

```txt
github.com/micoya/gocraft/cdao/<包名> 提供了实例的获取方式
```

例如需要获取gorm就是
```go
import github.com/micoya/gocraft/cdao/gormx

db := gormx.Must(*Dao) // 默认取config中对应dao配置的default实例
db := gormx.Must(*Dao, "micoya") // 取config中对应dao配置的micoya实例
```

**可用的Dao列表如下**

```go
redisx.Must(*Dao) *redis.Client // github.com/redis/go-redis/v9
gormx.Must(*Dao) *gorm.DB // gorm.io/gorm
ossx.Must(*Dao) *oss.Client // github.com/aliyun/aliyun-oss-go-sdk/oss
openaix.Must(*Dao) *openai.Client // github.com/openai/openai-go
```

## 目录结构

```
cmd/apiserver/main.go       # 入口，fx.New 启动
app/
  module.go           # fx 模块声明（Service / API等有app项目内依赖的)
  routes.go           # 路由注册，参数由 fx 注入
  api/                # Handler 层：参数绑定 + 调用 service + 响应封装
  service/            # 业务逻辑层：接收 Input struct，返回 model
  model/              # GORM 实体 + 响应 DTO
  common/response.go  # 统一 JSON 响应
  middleware/         # 中间件
pkg/                  # 包目录 通常该目录下的包和应用的MVC层不产生直接关系
  <package1>/
  <package2>/ 
config.example.yaml   # 配置文件模板
```

## 开发规范

- 使用MVC分层进行开发, 未要求的情况下不拆分Repo层等
- 功能需要编写单元测试
- Handler层应调用Serivce, 避免直接做业务操作
- 代码简洁不过度封装
- 带有IO的Service方法必须能够传递context并放到第一个位置
- 数据库操作均通过 `db.WithContext(ctx)` 传递 context
- 禁止在 service 层使用 `gin.Context` 的方法
- Handler和Service层的构造函数统一命名为 `NewXxxHandler` / `NewXxxService`
- Input struct 定义在 `service` 包中，如果Request Struct和Input不一致时在Handler的方法中建立Request Struct
- 文件名与资源名一致（小写，如 `user.go`）

**model层**
- GORM struct 和 DTO 放在一起，不引用 service/api 层
- 对于GORM struct至少需要完成`column`注解和`TableName()`方法, 并且提供必要的Migrate用字段

**fx 模块扩展**（新增资源 `Product` 为例）
1. 新建 `app/model/product.go`、`app/service/product.go`、`app/api/product.go`
2. `module.go` → `ServiceModule()` 追加 `service.NewProductService`，`APIModule()` 追加 `api.NewProductHandler`
3. `routes.go` → `RegisterRoutes` 参数追加 `*api.ProductHandler`，挂载路由

**响应格式**
- 业务成功：`common.OK(c, data)`
- 业务失败：`common.Fail(c, code, msg)`
- 客户端错误：`common.BadRequest(c, msg)` / `common.NotFound(c, msg)`
- 服务端错误：`common.InternalError(c, msg)`

**链路追踪**
每个包含IO的Service方法都需要完成otel链路追踪工作，方式是
```go
var orderServiceTracer = otel.Tracer("order-service")

func (s *OrderService) CreateOrder(ctx context.Context) error {
    ctx, span := orderServiceTracer.Start(ctx, "CreateOrder")
    defer span.End()
    // ...
}
```
项目已通过 `cotel.Provider` 和 `chttp.WithOtelProvider` 集成 OTel，HTTP 入口使用 otelgin 自动创建 span，Service 层需按上述方式显式创建子 span。
