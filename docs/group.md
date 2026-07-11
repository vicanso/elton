---
description: Group的相关方法说明
---

# Group

## NewGroup

创建一个路由组：路径前缀 + 组内公共中间件（非全局）。适用于同一资源前缀、统一鉴权等。返回的 `*Group` 提供 `GET`/`POST`/… 等方法（与 `Elton` 类似，且支持链式调用），最后用 `e.AddGroup` 一次性注册。

**2.0**：`g.NewGroup(path, handlers...)` 可创建**嵌套子组**（路径与中间件继承父组），子组随父组一起 `AddGroup`，无需再单独注册。

**Example**
```go
package main

import (
	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())
	// user相关的公共中间件
	noop := func(c *elton.Context) error {
		return c.Next()
	}

	userGroup := elton.NewGroup("/users", noop)
	userGroup.GET("/me", func(c *elton.Context) (err error) {
		// 从session中读取用户信息...
		c.Body = "user info"
		return
	}).POST("/login", func(c *elton.Context) (err error) {
		// 登录验证处理...
		c.Body = "login success"
		return
	})

	// 嵌套：/api + auth → /api/v1/users
	api := elton.NewGroup("/api", noop)
	v1 := api.NewGroup("/v1")
	v1.GET("/users", func(c *elton.Context) error {
		c.Body = "list"
		return nil
	})
	e.AddGroup(userGroup, api)

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```