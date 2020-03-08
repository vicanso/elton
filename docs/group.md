---
description: Group的相关方法说明
---

# Group

## NewGroup

创建一个组，它包括Path的前缀以及组内公共中间件（非全局），适用于创建有相同前置校验条件的路由处理，如用户相关的操作。返回的Group对象包括`GET`，`POST`，`PUT`等方法，与Elton类似，之后可以通过`AddGroup`将所有路由处理添加至Elton实例。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	responder "github.com/vicanso/elton-responder"
)

func main() {
	e := elton.New()

	e.Use(responder.NewDefault())

	noop := func(c *elton.Context) error {
		return c.Next()
	}

	userGroup := elton.NewGroup("/users", noop)
	userGroup.GET("/me", func(c *elton.Context) (err error) {
		// 从session中读取用户信息...
		c.Body = "user info"
		return
	})
	userGroup.POST("/login", func(c *elton.Context) (err error) {
		// 登录验证处理...
		c.Body = "login success"
		return
	})
	e.AddGroup(userGroup)

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```