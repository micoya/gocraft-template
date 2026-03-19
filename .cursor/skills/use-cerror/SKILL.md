---
name: use-cerror
description: 使用 cerror 统一业务错误处理，结构化错误码与 HTTP 状态码映射
---

## 概述

`cerror` 提供业务错误码体系，解决 Go 原生 `error` 接口缺乏结构化信息、HTTP 层无法自动映射状态码的问题。

```go
import "github.com/micoya/gocraft/cerror"
```

---

## 预定义错误码

```go
cerror.CodeBadRequest      // 400 - 参数错误
cerror.CodeUnauthorized    // 401 - 未登录
cerror.CodeForbidden       // 403 - 无权限
cerror.CodeNotFound        // 404 - 资源不存在
cerror.CodeConflict        // 409 - 资源冲突
cerror.CodeTooManyRequests // 429 - 频率限制
cerror.CodeInternal        // 500 - 内部错误
cerror.CodeUnavailable     // 503 - 服务不可用
```

---

## 创建错误

```go
// 使用预定义变量（最常见）
return cerror.ErrNotFound

// 自定义消息
return cerror.New(cerror.CodeNotFound, "用户不存在")

// 格式化消息
return cerror.Newf(cerror.CodeNotFound, "用户 %d 不存在", userID)

// 包装底层错误（保留原始错误链）
return cerror.Wrap(cerror.CodeInternal, "查询用户失败", err)
return cerror.Wrapf(cerror.CodeInternal, err, "查询用户 %d 失败", userID)
```

---

## 错误判断

```go
// 检查是否为指定 code
if cerror.IsCode(err, cerror.CodeNotFound) {
    // ...
}

// 提取 *cerror.Error（用于获取 code 和 message）
if ce, ok := cerror.FromError(err); ok {
    log.Printf("code=%d, message=%s", ce.Code(), ce.Message())
}

// 标准 errors.As 兼容
var ce *cerror.Error
if errors.As(err, &ce) {
    httpStatus := cerror.HTTPStatus(ce.Code())
}
```

---

## 在 Handler 层响应 cerror

`cerror` 与项目 `common` 包的响应函数配合使用：

```go
func (h *UserHandler) GetUser(c *gin.Context) {
    ctx := c.Request.Context()
    user, err := h.svc.GetUser(ctx, id)
    if err != nil {
        // 根据 cerror code 选择对应的响应函数
        if cerror.IsCode(err, cerror.CodeNotFound) {
            common.NotFound(c, err.Error())
            return
        }
        common.InternalError(c, err.Error())
        return
    }
    common.OK(c, user)
}
```

---

## 在 Service 层使用

```go
// service 层：用 cerror 包装底层错误，赋予语义
func (s *UserService) GetUser(ctx context.Context, id int64) (*model.User, error) {
    var user model.User
    err := s.db.WithContext(ctx).First(&user, id).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, cerror.Newf(cerror.CodeNotFound, "用户 %d 不存在", id)
    }
    if err != nil {
        return nil, cerror.Wrap(cerror.CodeInternal, "查询用户失败", err)
    }
    return &user, nil
}
```

---

## 自定义业务错误码

在业务模块内统一定义，业务 Code 从 `10000` 开始，每个模块持有 1000 个状态码：

```go
// app/common/errors.go 或各模块的 errors.go
const (
    // 订单模块 10000-10999
    CodeOrderNotPaid      cerror.Code = 10001
    CodeStockInsufficient cerror.Code = 10002
)

var (
    ErrOrderNotPaid      = cerror.New(CodeOrderNotPaid, "订单未支付")
    ErrStockInsufficient = cerror.New(CodeStockInsufficient, "库存不足")
)
```

在 Handler 中使用 `common.Fail(c, code, msg)` 响应自定义业务错误码：

```go
if errors.Is(err, ErrStockInsufficient) {
    common.Fail(c, int(CodeStockInsufficient), err.Error())
    return
}
```
