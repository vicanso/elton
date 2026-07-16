---
description: 路由参数
---

Elton 路由基于 Go 1.22+ 标准库 `net/http.ServeMux` 的 pattern 语法。命名参数用 `{name}`，剩余路径用 `{name...}`。取参使用 `c.Param(name)`（或 `c.Request.PathValue(name)` / `c.Params.ToMap()`）。

```go
package main

import (
	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {
	e := elton.New()
	e.Use(middleware.NewDefaultResponder())
	fn := func(c *elton.Context) (err error) {
		c.Body = c.Params.ToMap()
		return
	}
	e.GET("/books/{bookID}", fn)
	e.GET("/books/{bookID}/detail", fn)
	// 捕获 /books/summary/ 之后的剩余路径 → Param("path")
	e.GET("/books/summary/{path...}", fn)
	// 兼容旧写法：末尾 /* 会规范为 {path...}
	// e.GET("/books/summary/*", fn)
	e.GET("/books/trending/{year}/{month}/{day}", fn)
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## 语法说明

| pattern | 含义 |
|---|---|
| `/users/{id}` | 单段参数 |
| `/files/{path...}` | 匹配剩余路径（须在末尾） |
| `/{$}` | 仅匹配 `/`（注册 `"/"` 时框架会自动写成此形式） |
| `GET /users/{id}` | 内部按 method + path 注册；业务侧仍用 `e.GET(...)` |

**不支持**（注册时 panic）：

- 正则约束：`{id:[0-9]+}`
- 段内混合字面量与参数：`/books/{category}-{type}`、`/article/@{user}`

兼容转换：

- 段首 `:id` → `{id}`
- 路径末尾 `/*` → `/{path...}`
