---
name: use-cpager
description: 使用 cpager 分页工具，offset-based 分页参数解析与 GORM 集成
---

## 概述

`cpager` 提供 offset-based 分页的参数解析、归一化和泛型结果类型，深度集成 GORM。

```go
import "github.com/micoya/gocraft/cpager"
```

---

## 核心类型

```go
type Page struct { ... }  // 分页请求参数（page 从 1 开始）

type Result[T any] struct {
    Items      []T   `json:"items"`
    Total      int64 `json:"total"`
    Page       int   `json:"page"`
    PageSize   int   `json:"page_size"`
    TotalPages int   `json:"total_pages"`
    HasNext    bool  `json:"has_next"`
    HasPrev    bool  `json:"has_prev"`
}
```

**参数归一化规则：**

| 输入 | 归一化结果 |
|---|---|
| `page <= 0` | 自动设为 1 |
| `page_size <= 0` | 自动设为 20（默认值）|
| `page_size > 100` | 自动截断为 100（最大值）|

---

## 基本用法

### 在 Handler 中从 gin 请求解析

```go
func (h *UserHandler) ListUsers(c *gin.Context) {
    ctx := c.Request.Context()
    page := cpager.New(c.Query("page"), c.Query("page_size"))

    result, err := h.svc.ListUsers(ctx, page)
    if err != nil {
        common.InternalError(c, err.Error())
        return
    }
    common.OK(c, result)
}
```

### 在 Service 中使用 cpager.Paginate

```go
type UserService struct {
    db *gorm.DB
}

func (s *UserService) ListUsers(ctx context.Context, page cpager.Page) (*cpager.Result[model.User], error) {
    return cpager.Paginate[model.User](
        s.db.WithContext(ctx).
            Model(&model.User{}).
            Where("deleted_at IS NULL").
            Order("id DESC"),
        page,
    )
}
```

### 从整数创建（Service Input 中的分页参数）

```go
// Input struct 定义在 service 包
type ListUsersInput struct {
    Page     int
    PageSize int
}

func (s *UserService) ListUsers(ctx context.Context, input ListUsersInput) (*cpager.Result[model.User], error) {
    page := cpager.Of(input.Page, input.PageSize)
    return cpager.Paginate[model.User](
        s.db.WithContext(ctx).Model(&model.User{}).Order("id DESC"),
        page,
    )
}
```

---

## 响应格式示例

```json
{
  "items": [...],
  "total": 156,
  "page": 2,
  "page_size": 20,
  "total_pages": 8,
  "has_next": true,
  "has_prev": true
}
```

---

## 高级用法

### 手动使用 Scope（分步查询）

```go
page := cpager.Of(1, 20)

// 仅应用分页，不查 count
var items []model.User
s.db.WithContext(ctx).Scopes(page.Scope).Find(&items)

// 单独查 count
var total int64
s.db.WithContext(ctx).Model(&model.User{}).Where("status = ?", 1).Count(&total)
```

### 提前返回空结果

```go
if len(ids) == 0 {
    return cpager.Empty[model.User](page), nil
}
```

---

## 注意事项

- `Paginate` 内部使用 `Session(&gorm.Session{NewDB: true})` 执行 COUNT，避免影响原始查询的 SELECT 子句
- 返回的 `Items` 在无数据时为空切片 `[]T{}`（非 nil），JSON 序列化为 `[]` 而非 `null`
- `TotalPages` 最小值为 1（即使 total=0）
