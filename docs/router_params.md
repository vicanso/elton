---
description: 路由参数
---

Elton支持各种不同种类的路由参数配置形式，正则表达式或*等。需要注意的是，如果路由参数使正则，在参数不匹配时是无法获取对应的路由，导致接口404。

```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()
	e.Use(middleware.NewDefaultResponder())
	fn := func(c *elton.Context) (err error) {
		c.Body = c.Params.ToMap()
		return
	}
	e.GET("/books/{bookID:^[1-9][0-9]{0,3}$}", fn)
	e.GET("/books/{bookID:^[1-9][0-9]{0,3}$}/detail", fn)
	e.GET("/books/summary/*", fn)
	e.GET("/books/trending/{year}/{month}/{day}", fn)
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```
