# cod 

[![Build Status](https://img.shields.io/travis/vicanso/cod.svg?label=linux+build)](https://travis-ci.org/vicanso/cod)

Go web framework

开始接触后端开发是从nodejs开始，最开始使用的框架是express，后来陆续接触了其它的框架，觉得最熟悉简单的还是koa。使用golang做后端开发时，使用过gin，echo以及iris三个框架，它们的用法都比较类似（都支持中间件，中间件的处理与koa也类似）。但我还是用得不太习惯，不太习惯路由的响应处理，我更习惯koa的处理模式：出错返回error，正常返回body（body支持各种的数据类型）。

想着多练习golang，也想着自己去实现一套与koa更类似的web framework，因此则是cod的诞生。

```golang
package main

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/vicanso/cod"
	"github.com/vicanso/cod/middleware"
)

func main() {

	d := cod.New()

	d.Use(middleware.NewRecover())

	// 针对出错error生成相应的HTTP响应数据（http状态码以及响应数据）
	d.Use(middleware.NewDefaultErrorHandler()))

	d.Use(middleware.NewStats(middleware.StatsConfig{
		// 返回接口处理时长、状态码等
		OnStats: func(stats *middleware.StatsInfo, _ *cod.Context) {
			log.Println("stats:", stats)
		},
	}))

	// 请求处理时长
	d.Use(func(c *cod.Context) (err error) {
		started := time.Now()
		err = c.Next()
		log.Printf("response time:%s", time.Since(started))
		return
	})

	// 只允许使用json形式提交参数，以及长度限制为10KB
	d.Use(middleware.NewDefaultBodyParser())

	// fresh与etag，fresh在etag前添加
	d.Use(middleware.NewDefaultFresh())
	d.Use(middleware.NewDefaultETag())

	d.Use(middleware.NewCompress(middleware.CompressConfig{
		// 最小压缩长度设置为1（测试需要，实际可根据实际场景配置或不配置）
		MinLength: 1,
	}))

	// 指定使用querystring中的fields来筛选响应数据
	d.Use(middleware.NewJSONPicker(middleware.JSONPickerConfig{
		Field: "fields",
	}))

	// 根据Body生成相应的HTTP响应数据
	d.Use(middleware.NewDefaultResponder()))


	d.GET("/users/me", func(c *cod.Context) (err error) {
		c.Body = &struct {
			Name string `json:"name"`
			Type string `json:"type"`
		}{
			"tree.xie",
			"vip",
		}
		return
	})

	loginTracker := func(info *middleware.TrackerInfo, _ *cod.Context) {
		// 输出track日志，在实际使用中可以记录至数据库等
		fmt.Println("login:", info)
	}
	d.POST("/users/login", middleware.NewTracker(middleware.TrackerConfig{
		OnTrack: loginTracker,
		// 指定哪些字段做***处理
		Mask: regexp.MustCompile(`password`),
	}), func(c *cod.Context) (err error) {
		c.Body = &struct {
			Account string `json:"account"`
		}{
			"tree.xie",
		}
		return
	})
	d.ListenAndServe(":8001")
}
```

上面的例子已经实现了简单的HTTP响应（得益于golang自带http的强大），整个框架中主要有两个struct：Cod与Context，下面我们来详细介绍这两个struct。

## Cod

[cod.Cod](./docs/cod.md)

## Group

### NewGroup 

创建一个组，它包括Path的前缀以及组内公共中间件（非全局），适用于创建有相同前置校验条件的路由处理，如用户相关的操作。返回的Group对象包括`GET`，`POST`，`PUT`等方法，与Cod的似，之后可以通过`AddGroup`将所有路由处理添加至cod实例。

```go
userGroup := cod.NewGroup("/users", noop)
userGroup.GET("/me", func(c *cod.Context) (err error) {
	// 从session中读取用户信息...
	c.Body = "user info"
	return
})
userGroup.POST("/login", func(c *cod.Context) (err error) {
	// 登录验证处理...
	c.Body = "login success"
	return
})
```