# gocraft-template

基于 [gocraft](https://github.com/micoya/gocraft) 框架的 Go 项目样板工程。

## 技术栈

| 层次 | 技术 |
|---|---|
| HTTP 框架 | [gin](https://github.com/gin-gonic/gin) |
| 依赖注入 | [uber/fx](https://github.com/uber-go/fx) |
| 配置 | gocraft/config（viper + godotenv） |
| 日志 | gocraft/clog（log/slog 封装） |

> 模板为 demo 模式，user 等接口返回固定数据，无数据库依赖，可直接运行。

## 项目结构

```
gocraft-template/
├── cmd/
│   └── apiserver/
│       └── main.go         # 程序入口，fx 启动
├── app/
│   ├── module.go           # fx 依赖注入模块声明（Infra / Service / API）
│   ├── routes.go           # 路由注册
│   ├── api/                # HTTP Handler 层，只做参数绑定和响应封装
│   │   ├── hello.go
│   │   └── user.go
│   ├── service/            # 业务逻辑层
│   │   ├── hello.go
│   │   └── user.go
│   ├── model/              # 数据模型（GORM 实体 + DTO）
│   │   ├── hello.go
│   │   └── user.go
│   └── common/
│       └── response.go     # 统一 JSON 响应格式
├── config.example.yaml     # 配置模板（入库）
├── config.yaml             # 本地配置（不入库）
└── go.mod
```

## 快速开始

```bash
cp config.example.yaml config.yaml
go run ./cmd/apiserver
```

模板为 demo 模式，无需数据库，user 接口返回固定数据，可直接运行。

## API 接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/healthz` | 健康检查 |
| GET | `/api/v1/hello?name=xxx` | Hello 示例 |
| GET | `/api/v1/users` | 用户列表 |
| POST | `/api/v1/users` | 创建用户 |
| GET | `/api/v1/users/:id` | 获取单个用户 |
| PATCH | `/api/v1/users/:id` | 更新用户 |
| DELETE | `/api/v1/users/:id` | 删除用户 |

### 响应格式

所有接口统一返回以下结构：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

`code` 为 `0` 表示成功，非零表示失败（约定与 HTTP 状态码对应）。

## 扩展指南

### 新增一个资源（如 Product）

1. **模型**：在 `app/model/product.go` 定义 `Product` struct
2. **服务**：在 `app/service/product.go` 实现业务逻辑（demo 模式可返回固定数据；接入真实 DB 时注入 `*gorm.DB`）
3. **Handler**：在 `app/api/product.go` 实现 HTTP handler，构造函数注入 `*service.ProductService`
4. **注册**：
   - `app/module.go` → `ServiceModule()` 和 `APIModule()` 中 `fx.Provide` 追加对应构造函数
   - `app/routes.go` → `RegisterRoutes` 参数追加 `*api.ProductHandler`，挂载路由

### 新增 cmd（如 worker）

在 `cmd/worker/main.go` 中使用 `app.ServiceModule()`，无需启动 HTTP 服务器。

## 配置说明

环境变量优先级高于配置文件，层级分隔符为 `__`：

```bash
# 等价于 config.yaml 中 http_server.addr
export HTTP_SERVER__ADDR=":9090"
```

支持 `.env` 文件（项目根目录），加载后可被系统环境变量覆盖。
