# Elton 

[![license](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/vicanso/elton/blob/master/LICENSE)
[![Build Status](https://github.com/vicanso/elton/workflows/Test/badge.svg)](https://github.com/vicanso/elton/actions)

![Alt](https://repobeats.axiom.co/api/embed/4f64b99db39c6a75b6980ebb3c756244b246a718.svg "Repobeats analytics image")

Elton的实现参考了[koa](https://github.com/koajs/koa)以及[echo](https://github.com/labstack/echo)，中间件的调用为洋葱模型：请求由外至内，响应由内至外。主要特性如下：

- 处理函数（中间件）均以返回error的形式响应出错，方便使用统一的出错处理中间件将出错统一转换为对应的输出（JSON），并根据出错的类型等生成各类统计分析
- 成功响应数据直接赋值至Context.Body（any），由统一的响应中间件将其转换为对应的输出（JSON，XML）
- 支持不同种类的事件，如`OnBefore`、`OnDone`、`OnError`等，方便添加各类统计行为

如何使用`elton`开发WEB后端程序，可以参考[一步一步学习如何使用elton](https://treexie.gitbook.io/elton-beginner/)

## 安装

```bash
go get github.com/vicanso/elton/v2
```

从 1.x 升级请参考[2.0 迁移指南](./docs/migration-v2.md)。

## Hello, World!

下面我们来演示如何使用`elton`返回`Hello, World!`，并且添加了一些常用的中间件。

```go
package main

import (
	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {
	e := elton.New()

	// Recover + Error + RequestID + BodyParser + Fresh + ETag + Responder
	e.Use(middleware.Recommended()...)
	// 可选：e.Use(middleware.NewDefaultTimeout(5*time.Second))
	// 可选：e.Use(middleware.NewDefaultCORS())

	e.GET("/", func(c *elton.Context) error {
		c.Body = &struct {
			Message string `json:"message,omitempty"`
		}{
			"Hello, World!",
		}
		return nil
	})

	e.GET("/books/{id}", func(c *elton.Context) error {
		c.Body = &struct {
			ID string `json:"id,omitempty"`
		}{
			c.Param("id"),
		}
		return nil
	})

	e.POST("/login", func(c *elton.Context) error {
		c.SetContentTypeByExt(".json")
		c.Body = c.RequestBody
		return nil
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

```bash
go run ./examples/hello
# 或自建 main.go 后 go run .
```

之后在浏览器中打开`http://localhost:3000/`则能看到返回的`Hello, World!`。更多示例见 [`examples/`](./examples/)。

## 路由

elton每个路由可以添加多个中间件处理函数，根据路由与及HTTP请求方法指定不同的路由处理函数。而全局的中间件则可通过`Use`方法来添加。

```go
e.Use(...func(*elton.Context) error)
e.Method(path string, ...func(*elton.Context) error)
```

- `e` 为`elton`实例化对象
- `Method` 为HTTP的请求方法，如：`GET`, `PUT`, `POST`等等
- `path` 为HTTP路由路径
- `func(*elton.Context) error` 为路由处理函数（中间件），当匹配的路由被请求时，对应的处理函数则会被调用

### 路由示例

路由使用 Go 1.22+ 标准库 [`net/http.ServeMux`](https://pkg.go.dev/net/http#ServeMux) 的 pattern（`{name}`、`{name...}`、方法匹配）。

```go
// 带参数路由
e.GET("/users/{type}", func(c *elton.Context) error {
	c.BodyBuffer = bytes.NewBufferString(c.Param("type"))
	return nil
})

// 捕获剩余路径
e.GET("/files/{path...}", func(c *elton.Context) error {
	c.BodyBuffer = bytes.NewBufferString(c.Param("path"))
	return nil
})

// 带中间件的路由配置
e.GET("/users/me", func(c *elton.Context) error {
	c.Set("account", "tree.xie")
	return c.Next()
}, func(c *elton.Context) error {
	c.BodyBuffer = bytes.NewBufferString(elton.GetContextValue[string](c, "account"))
	return nil
})
```

## 中间件

简单方便的中间件机制，依赖各类定制的中间件，通过各类中间件的组合，方便快捷实现各类HTTP服务，简单介绍数据响应与出错处理的中间件。需要注意，elton中默认不会执行所有的中间件，每个中间件决定是否需要执行后续处理，如果需要则调用Next()函数，与gin不一样(gin默认为执行所有，若不希望执行后续的中间件，则调用Abort)。

### responder

HTTP请求响应数据时，需要将数据转换为Buffer返回，而在应用时响应数据一般为各类的struct或map等结构化数据，因此elton提供了`Body`（`any`）字段来保存这些数据，再由中间件转换为对应的字节数据。内置的 `middleware.NewDefaultResponder()` 会将 struct/map 转为 JSON 并设置 `Content-Type`，对 `string`/`[]byte` 则直接输出。

```go
package main

import (
	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {

	e := elton.New()
	// 对响应数据 c.Body 转换为相应的json响应
	e.Use(middleware.NewDefaultResponder())

	getSession := func(c *elton.Context) error {
		c.Set("account", "tree.xie")
		return c.Next()
	}
	e.GET("/users/me", getSession, func(c *elton.Context) (err error) {
		c.Body = &struct {
			Name string `json:"name"`
			Type string `json:"type"`
		}{
			elton.GetContextValue[string](c, "account"),
			"vip",
		}
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

### error

当请求处理失败时，直接返回error则可，elton从error中获取出错信息并输出。默认的出错处理并不适合实际应用场景，建议使用自定义出错类配合中间件，便于统一的错误处理，程序监控，下面是引入错误中间件将出错转换为json形式的响应。

```go
package main

import (
	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
	"github.com/vicanso/hes"
)

func main() {

	e := elton.New()
	// 指定出错以json的形式返回
	e.Use(middleware.NewError(middleware.ErrorConfig{
		ResponseType: "json",
	}))

	e.GET("/", func(c *elton.Context) (err error) {
		err = &hes.Error{
			StatusCode: 400,
			Category:   "users",
			Message:    "出错啦",
		}
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

更多的中间件可以参考[middlewares](./docs/middlewares.md)

## bench

```bash
go test -bench=. -benchmem ./...
```

参考结果（darwin/arm64，elton 2.0 / Go 1.24+）：

```
goos: darwin
goarch: arm64
pkg: github.com/vicanso/elton/v2
BenchmarkRoutes-12                 	18873722	        55.69 ns/op	     120 B/op	       2 allocs/op
BenchmarkGetFunctionName-12        	295134098	         4.082 ns/op	       0 B/op	       0 allocs/op
BenchmarkContextGet-12             	57954447	        20.43 ns/op	       0 B/op	       0 allocs/op
BenchmarkContextNewMap-12          	227259025	         5.333 ns/op	       0 B/op	       0 allocs/op
BenchmarkConvertServerTiming-12    	 3219936	       367.6 ns/op	     360 B/op	      11 allocs/op
BenchmarkStatus-12                 	1000000000	         0.2438 ns/op	       0 B/op	       0 allocs/op
BenchmarkFresh-12                  	 2878122	       407.8 ns/op	     326 B/op	       8 allocs/op
BenchmarkStatic-12                 	   64177	     17887 ns/op	   21147 B/op	     471 allocs/op
BenchmarkGitHubAPI-12              	   40027	     29236 ns/op	   26832 B/op	     609 allocs/op
BenchmarkGplusAPI-12               	  837277	      1295 ns/op	    1744 B/op	      39 allocs/op
BenchmarkParseAPI-12               	  445596	      2526 ns/op	    3479 B/op	      78 allocs/op
BenchmarkRWMutexSignedKeys-12      	313402153	         3.828 ns/op	       0 B/op	       0 allocs/op
BenchmarkAtomicSignedKeys-12       	1000000000	         0.7715 ns/op	       0 B/op	       0 allocs/op
BenchmarkReadAllInitCap-12         	     520	   2258030 ns/op	73546877 B/op	      14 allocs/op
PASS
ok  	github.com/vicanso/elton/v2
goos: darwin
goarch: arm64
pkg: github.com/vicanso/elton/v2/middleware
BenchmarkBodyParserBufferPool-12    	  219397	      5430 ns/op	   23393 B/op	      24 allocs/op
BenchmarkGenETag-12                 	  878080	      1370 ns/op	     128 B/op	       5 allocs/op
BenchmarkMd5-12                     	  251551	      4744 ns/op	      96 B/op	       5 allocs/op
BenchmarkNewShortHTTPHeader-12      	23356375	        51.05 ns/op	      80 B/op	       2 allocs/op
BenchmarkNewHTTPHeader-12           	17233610	        69.85 ns/op	      88 B/op	       3 allocs/op
BenchmarkNewHTTPHeaders-12          	 1528754	       797.2 ns/op	    1136 B/op	      23 allocs/op
BenchmarkHTTPHeaderMarshal-12       	 1501634	       797.6 ns/op	     920 B/op	      16 allocs/op
BenchmarkToHTTPHeader-12            	 1452776	       842.2 ns/op	    1272 B/op	      34 allocs/op
BenchmarkHTTPHeaderUnmarshal-12     	  513957	      2328 ns/op	    1216 B/op	      36 allocs/op
BenchmarkLRUStore-12                	13680637	        87.71 ns/op	      16 B/op	       1 allocs/op
BenchmarkProxy-12                   	   25860	     45085 ns/op	   20476 B/op	     112 allocs/op
PASS
ok  	github.com/vicanso/elton/v2/middleware
```
