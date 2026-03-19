---
name: create-resource
description:  创建 Model, Service, Handler, fx(依赖注入) 注册, 路由挂载
---

## 说明

本 Skill 描述在 gocraft-template 项目中新增一个完整业务资源（MVC 三层 + fx 注册）的标准流程。以新增 `Order` 资源为例，将资源名替换成实际名称即可。

如果需求不需要创建完整的三层, 请根据需求灵活处理。

---

## 步骤一：创建 Model

新建 `app/model/order.go`，GORM struct 与响应 DTO 放在一起，不引用 service/api 层。

```go
package model

import "time"

type Order struct {
    ID        int64     `gorm:"column:id;primaryKey" json:"id"`
    UserID    int64     `gorm:"column:user_id;not null" json:"user_id"`
    Amount    float64   `gorm:"column:amount;not null" json:"amount"`
    Status    string    `gorm:"column:status;not null" json:"status"`
    CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Order) TableName() string {
    return "orders"
}

// MigrateFields 用于 AutoMigrate，返回模型实例
func MigrateFields() []any {
    return []any{&Order{}}
}
```

**模型创建要点**

- GORM模型需要必须声明column和用于migrate的信息, 必须实现`TableName()`方法, 避免框架内置逻辑判断名字
- 隐私字段例如`password`等应当默认加入注解`json:"-"`
- 如果需要 AutoMigrate，在应用启动时传入模型实例（由 `cfx.CoreModule` 或自定义启动逻辑管理）

---

## 步骤二：创建 Service

新建 `app/service/order.go`，Input struct 定义在 service 包中，IO 方法第一个参数必须是 `context.Context`，每个 IO 方法都需完成 OTel span。

```go
package service

import (
    "context"
    "log/slog"

    "go.opentelemetry.io/otel"
    "gorm.io/gorm"

    "gocraft-template/app/model"
    "github.com/micoya/gocraft/cdao"
    "github.com/micoya/gocraft/cdao/gormx"
)

var orderServiceTracer = otel.Tracer("order-service")

type CreateOrderInput struct {
    UserID int64
    Amount float64
}

type OrderService struct {
    db  *gorm.DB
    log *slog.Logger
}

func NewOrderService(dao *cdao.DAO, log *slog.Logger) *OrderService {
    return &OrderService{
        db:  gormx.Must(dao),
        log: log,
    }
}

// CreateOrder 创建订单
func (s *OrderService) CreateOrder(ctx context.Context, input CreateOrderInput) (*model.Order, error) {
    ctx, span := orderServiceTracer.Start(ctx, "CreateOrder")
    defer span.End()

    order := &model.Order{
        UserID: input.UserID,
        Amount: input.Amount,
        Status: "pending",
    }
    if err := s.db.WithContext(ctx).Create(order).Error; err != nil {
        return nil, err
    }
    return order, nil
}

// GetOrder 根据 ID 获取订单
func (s *OrderService) GetOrder(ctx context.Context, id int64) (*model.Order, error) {
    ctx, span := orderServiceTracer.Start(ctx, "GetOrder")
    defer span.End()

    var order model.Order
    if err := s.db.WithContext(ctx).First(&order, id).Error; err != nil {
        return nil, err
    }
    return &order, nil
}
```

**注意：**
- 如果不需要数据库，可以注入 `*slog.Logger` 等其他依赖代替 `*cdao.Dao`
- 禁止在 service 层使用 `gin.Context` 的任何方法

---

## 步骤三：创建 Handler

新建 `app/api/order.go`，Handler 只做参数绑定、调用 service、封装响应。Request struct 与 Input 不一致时在方法内定义 Request struct。

```go
package api

import (
    "log/slog"

    "github.com/gin-gonic/gin"

    "gocraft-template/app/common"
    "gocraft-template/app/service"
)

type OrderHandler struct {
    svc *service.OrderService
    log *slog.Logger
}

func NewOrderHandler(svc *service.OrderService, log *slog.Logger) *OrderHandler {
    return &OrderHandler{svc: svc, log: log}
}

// CreateOrder POST /api/v1/orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
    ctx := c.Request.Context()

    var req struct {
        UserID int64   `json:"user_id" binding:"required"`
        Amount float64 `json:"amount" binding:"required,gt=0"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        common.BadRequest(c, err.Error())
        return
    }

    order, err := h.svc.CreateOrder(ctx, service.CreateOrderInput{
        UserID: req.UserID,
        Amount: req.Amount,
    })
    if err != nil {
        h.log.ErrorContext(ctx, "create order failed", slog.Any("error", err))
        common.InternalError(c, err.Error())
        return
    }

    common.OK(c, order)
}

// GetOrder GET /api/v1/orders/:id
func (h *OrderHandler) GetOrder(c *gin.Context) {
    ctx := c.Request.Context()

    var uri struct {
        ID int64 `uri:"id" binding:"required"`
    }
    if err := c.ShouldBindUri(&uri); err != nil {
        common.BadRequest(c, err.Error())
        return
    }

    order, err := h.svc.GetOrder(ctx, uri.ID)
    if err != nil {
        common.NotFound(c, "order not found")
        return
    }

    common.OK(c, order)
}
```

---

## 步骤四：注册到 fx

编辑 `cmd/apiserver/main.go`，在对应位置追加新依赖：

```go
fx.New(
    fx.WithLogger(cfx.LoggerProvider),

    cfx.CoreModule[struct{}](),
    cfx.DAOModule[struct{}](),

    fx.Provide(
        service.NewHelloService,
        service.NewOrderService, // 追加
    ),

    fx.Provide(/* *chttp.Server ... */),

    fx.Provide(
        api.NewHelloHandler,
        api.NewOrderHandler, // 追加
    ),
    fx.Invoke(app.RegisterRoutes),
).Run()
```

---

## 步骤五：挂载路由

编辑 `app/routes.go`，在 `RegisterRoutes` 参数中追加新 Handler 并挂载路由：

```go
func RegisterRoutes(
    server *chttp.Server,
    helloHandler *api.HelloHandler,
    orderHandler *api.OrderHandler, // 追加
) {
    server.Engine().GET("/", helloHandler.Welcome)

    v1 := server.Engine().Group("/api/v1")
    {
        v1.GET("/hello", helloHandler.SayHello)

        // 订单路由
        v1.POST("/orders", orderHandler.CreateOrder)
        v1.GET("/orders/:id", orderHandler.GetOrder)
    }
}
```

---

## 响应格式速查

| 场景       | 调用方式                          |
|------------|-----------------------------------|
| 成功       | `common.OK(c, data)`              |
| 业务失败   | `common.Fail(c, code, msg)`       |
| 参数错误   | `common.BadRequest(c, msg)`       |
| 资源不存在 | `common.NotFound(c, msg)`         |
| 服务器错误 | `common.InternalError(c, msg)`    |

---

## 文件清单

新增一个资源需要改动的文件：

```
app/model/order.go           # 新建
app/service/order.go         # 新建
app/api/order.go             # 新建
cmd/apiserver/main.go        # 修改：fx.Provide 追加 Service + Handler
app/routes.go                # 修改：RegisterRoutes 追加参数 + 路由
```
