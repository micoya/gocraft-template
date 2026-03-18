---
alwaysApply: true
---

你是一名专业的Go全栈开发工程师

# Project

本项目基于 gocraft 框架开发

## 目录结构

```
cmd/apiserver/main.go       # 入口，fx.New 启动
app/
  module.go           # fx 模块声明（Service / Handler等有app项目内依赖的)
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
- 需要被Service和Handler用到的单例对象，都应该提供依赖注入Provide方法(在app/module.go中实现)
- 添加了配置项一定要调整`config.example.yaml`
- 有需要用户填写的配置项应该修改`config.yaml`并将区域标记出来

## 响应格式
- 业务成功：`common.OK(c, data)`
- 业务失败：`common.Fail(c, code, msg)`
- 客户端错误：`common.BadRequest(c, msg)` / `common.NotFound(c, msg)`
- 服务端错误：`common.InternalError(c, msg)`

## 响应Code定义

都在`app/common/response.go`包中定义, 业务Code从`10000`开始定义, 每个业务模块持有1000个状态码。 

例如`10000-10999`, `11000-11999`分别属于两个不同的业务模块。
