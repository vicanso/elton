# cod 

[![Build Status](https://img.shields.io/travis/vicanso/cod.svg?label=linux+build)](https://travis-ci.org/vicanso/cod)

Go web framework

开始接触后端开发是从nodejs开始，最开始使用的框架是express，后来陆续接触了其它的框架，觉得最熟悉简单的还是koa。使用golang做后端开发时，使用过gin，echo以及iris三个框架，它们的用法都比较类似（都支持中间件，中间件的处理与koa也类似）。但我还是用得不太习惯，不太习惯路由的响应响应，我更习惯koa的处理模式：出错返回error，正常返回body（body支持各种的数据类型）。

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

实现HTTP服务的监听、中间件的顺序调用以及路由的选择调用。

### Server

http.Server对象，在初始化Cod时，将创建一个默认的Server，可以再根据自己的应用场景调整Server的参数配置，如下：

```go
d := cod.New()
d.Server.MaxHeaderBytes = 10 * 1024
```

### Router

[httprouter.Router](https://github.com/julienschmidt/httprouter)对象，Cod使用httprouter来处理http的路由于处理函数的关系，此对象如无必要无需要做调整。

### Routers

记录当前Cod实例中所有的路由信息，为[]*RouterInfo，每个路由信息包括Method与Path，此属性只用于统计等场景使用，不需要调整。

```go
// RouterInfo router's info
RouterInfo struct {
  Method string `json:"method,omitempty"`
  Path   string `json:"path,omitempty"`
}
```

### Middlewares

当前Cod实例中的所有中间件处理函数，为[]Handler，如果需要添加中间件，尽量使用Use，不要直接append此属性。

```go
d := cod.New()
d.Use(middleware.NewResponder(middleware.ResponderConfig{}))
```

### ErrorHandler

自定义的Error处理，若路由处理过程中返回Error，则会触发此调用，非未指定此处理函数，则使用默认的处理。

注意若在处理过程中返回的Error已被处理（如middleware.NewResponder），则并不会触发此出错调用，尽量使用NewResponder将出错转换为出错的HTTP响应。

```go
d := cod.New()

d.ErrorHandler = func(c *cod.Context, err error) {
  if err != nil {
    log.Printf("未处理异常，url:%s, err:%v", c.Request.RequestURI, err)
  }
  c.Response.WriteHeader(http.StatusInternalServerError)
  c.Response.Write([]byte(err.Error()))
}

d.GET("/ping", func(c *cod.Context) (err error) {
  return errors.New("abcd")
})
d.ListenAndServe(":8001")
```

### NotFoundHandler

未匹配到相应路由时的处理，当无法获取到相应路由时，则会调用此函数（未匹配相应路由时，所有的中间件也不会被调用）。如果有相关统计需要或者自定义的404页面，则可调整此函数，否则可不设置（使用默认）。

```go
d := cod.New()

d.NotFoundHandler = func(resp http.ResponseWriter, req *http.Request) {
  // 要增加统计，方便分析404的处理是被攻击还是接口调用错误
  resp.WriteHeader(http.StatusNotFound)
  resp.Write([]byte("Not found"))
}

d.GET("/ping", func(c *cod.Context) (err error) {
  return errors.New("abcd")
})
d.ListenAndServe(":8001")
```

### GenerateID

ID生成函数，用于每次请求调用时，生成唯一的ID值。

```go
d := cod.New()

d.GenerateID = func() string {
  t := time.Now()
  entropy := rand.New(rand.NewSource(t.UnixNano()))
  return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.GET("/ping", func(c *cod.Context) (err error) {
  log.Println(c.ID)
  c.Body = "pong"
  return
})
d.ListenAndServe(":8001")
```