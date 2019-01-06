# cod 

[![Build Status](https://img.shields.io/travis/vicanso/cod.svg?label=linux+build)](https://travis-ci.org/vicanso/cod)

Go web framework

开始接触后端开发是从nodejs开始，最开始使用的框架是express，后来陆续接触了其它的框架，觉得最熟悉简单的还是koa。使用golang做后端开发时，使用过gin，echo以及iris三个框架，它们的用法都比较类似（都支持中间件，中间件的处理与koa也类似）。但我还是用得不太习惯，不太习惯路由的响应处理，我更习惯koa的处理模式：出错返回error，正常返回body（body支持各种的数据类型）。

想着多练习golang，也想着自己去实现一套与koa更类似的web framework，因此则是cod的诞生。

```golang
package main

import (
	"log"
	"time"

	"github.com/vicanso/cod"
	"github.com/vicanso/cod/middleware"
)

func main() {
	d := cod.New()

	// 请求处理时长
	d.Use(func(c *cod.Context) (err error) {
		started := time.Now()
		err = c.Next()
		log.Printf("response time:%s", time.Since(started))
		return
	})

	// 针对出错error生成相应的HTTP响应数据（http状态码以及响应数据）
	// 或者成功处理的Body生成相应的HTTP响应数据
	d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

	// 路由处理函数，响应pong字符串
	d.GET("/ping", func(c *cod.Context) (err error) {
		c.Body = "pong"
		return
	})
	d.ListenAndServe(":8001")
}
```

上面的例子已经实现了简单的HTTP响应（得益于golang自带http的强大），整个框架中主要有两个struct：Cod与Context，下面我们来详细介绍这两个struct。

## Cod

[cod.Cod](./docs/cod.md)
