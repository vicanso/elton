---
description: 路由参数
---

Elton支持各种不同种类的路由参数配置形式，正则表达式或*等。

```go
package main

import (
	"github.com/vicanso/elton"
	responder "github.com/vicanso/elton-responder"
)

func main() {
	e := elton.New()
	e.Use(responder.NewDefault())
	e.GET("/{bookID:^[1-9][0-9]{0,3}$}", func(c *elton.Context) (err error) {
		c.Body = &struct {
			Name string `json:"name,omitempty"`
		}{
			"代码大全",
		}
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```
