# gocraft-template

基于 [gocraft](https://github.com/micoya/gocraft) 框架的 Go 项目样板工程。

给 `Micoya` 快速创建 Go + Gin + fx + gocraft 项目用。

## 快速开始

```bash
go install github.com/micoya/gocraft/cmd/gocraft@latest

gocraft new <my-project>
```

进入项目后：

```bash
go mod tidy
go test ./...
go run ./cmd/apiserver
```

默认 HTTP 地址为 `:8080`，健康检查路径为 `/healthz`。本地敏感配置写入 `config.yaml`，不要提交；可从 `config.example.yaml` 复制后按环境调整。

## 目录结构

```txt
cmd/apiserver/main.go       # fx 启动入口
app/
  module.go                 # 业务依赖注入入口
  routes.go                 # HTTP 路由注册
  api/                      # Handler 层
  service/                  # 业务逻辑层
  model/                    # GORM Model / DTO
  common/response.go        # 统一响应封装
pkg/                        # 与 MVC 层解耦的通用包
config.example.yaml         # 配置样例与运维对照
```

## 开发约定

- 基础设施由 `cmd/*/main.go` 通过 `cfx.Provide*` 注册，业务侧依赖集中放在 `app.Module()`。
- Handler 只做参数绑定、调用 Service、返回 `common.OK/Fail/...`。
- 带 IO 的 Service 方法第一个参数使用 `context.Context`，数据库操作使用 `db.WithContext(ctx)`。
- 新增配置项时同步更新 `config.example.yaml`；需要本地实填的敏感配置写入 `config.yaml`。
- 南北向接口注册到 `/api`，东西向内部接口注册到 `/internal-api`。

## AI 开发支持

模板内置两套 AI 辅助说明：

- Cursor：`.cursor/rules/project.mdc` 与 `.cursor/skills/*/SKILL.md`
- Codex：`AGENTS.md` 与 `.codex/skills/*/SKILL.md`

两套 skill 内容保持一致，覆盖常见 gocraft 能力：DAO、fx 注入、cuid、ccache、ccron、clocker、climiter、cpager、cworker、cpubsub、cerror 以及业务资源创建。
