# gocraft-template

本项目基于 gocraft 框架开发，当前模板依赖 `github.com/micoya/gocraft v0.3.6`。

## 目录结构

```txt
cmd/apiserver/main.go       # 入口，fx.New 启动
cmd/xxxxx/                  # 其他启动入口应用
app/
  module.go                 # fx 模块声明（Service / Handler 等业务依赖）
  routes.go                 # 路由注册，参数由 fx 注入
  api/                      # Handler 层：参数绑定 + 调用 service + 响应封装
  service/                  # 业务逻辑层：接收 Input struct，返回 model
  model/                    # GORM 实体 + 响应 DTO
  common/response.go        # 统一 JSON 响应
  middleware/               # 中间件
pkg/                        # 与应用 MVC 层解耦的通用包
config.example.yaml         # 配置文件模板
```

## Codex 工作约定

- 处理 gocraft 能力时，优先读取 `.codex/skills/*/SKILL.md` 中对应 skill，例如 `use-dao`、`use-cuid`、`use-fx-provide`。
- 修改 `.cursor/skills` 后，同步更新 `.codex/skills` 中同名 skill，避免 Cursor 与 Codex 说明漂移。
- 新增配置项时同步更新 `config.example.yaml`；需要用户本地填写的敏感项写入 `config.yaml`，不要提交。

## 开发规范

- 使用 MVC 分层开发，未要求时不拆分 Repository 层。
- 功能需要编写单元测试。
- Handler 层应调用 Service，避免直接做业务操作；不要直接拿 DAO 做业务读写，除非只是非常薄的探活/内部调试端点。
- 代码简洁，不过度封装。
- 带 IO 的 Service 方法必须把 `context.Context` 放在第一个参数。
- 数据库操作均通过 `db.WithContext(ctx)` 传递 context。
- 禁止在 service 层使用 `gin.Context` 的方法。
- Handler 和 Service 构造函数统一命名为 `NewXxxHandler` / `NewXxxService`。
- Input struct 定义在 `service` 包中；如果 Request struct 和 Input 不一致，在 Handler 方法中建立 Request struct。
- 文件名与资源名一致，小写，例如 `user.go`。
- 需要被 Service 和 Handler 用到的单例对象，都应该在 `app/module.go` 提供依赖注入。
- 给前端、客户端使用的南北向路由注册到 `/api`。
- 给集群内部调用的东西向路由注册到 `/internal-api`，默认无需鉴权。
- 接口不使用 RESTful 风格，使用 `/版本/业务域/动作`，动作使用小写下划线。

## gocraft 常用接入点

- 基础设施在 `cmd/*/main.go` 用 `cfx.ProvideConfig`、`ProvideLogger`、`ProvideOtel`、`ProvideDAO`、`ProvideHTTPServer` 等注册。
- 业务依赖在 `app/module.go` 用 `fx.Provide` 注册；不要为业务包新增全局 cfx 封装。
- 使用 DAO 资源时，在 `main` 包 blank import 对应 `github.com/micoya/gocraft/cdao/provider/...`。
- Redis 短雪花使用 `cuid.Shortflake` 或 `cfx.ProvideUIDShortflake("default", opts...)`，它依赖 `dao.redis`，但没有 `uid.shortflake` 配置块。

## 响应格式

- 业务成功：`common.OK(c, data)`
- 业务失败：`common.Fail(c, code, msg)`
- 客户端错误：`common.BadRequest(c, msg)` / `common.NotFound(c, msg)`
- 服务端错误：`common.InternalError(c, msg)`

## 响应 Code

业务 Code 在 `app/common/response.go` 中定义，从 `10000` 开始，每个业务模块持有 1000 个状态码。

例如 `10000-10999`、`11000-11999` 分别属于两个不同业务模块。
