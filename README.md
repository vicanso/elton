# Elton 

[![license](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/vicanso/elton/blob/master/LICENSE)
[![Build Status](https://github.com/vicanso/elton/workflows/Test/badge.svg)](https://github.com/vicanso/elton/actions)


Elton的实现参考了[koa](https://github.com/koajs/koa)以及[echo](https://github.com/labstack/echo)，中间件的调用为洋葱模型：请求由外至内，响应由内至外。主要特性如下：

- 处理函数（中间件）均以返回error的形式响应出错，方便使用统一的出错处理中间件将出错统一转换为对应的输出（JSON），并根据出错的类型等生成各类统计分析
- 成功响应数据直接赋值至Context.Body（interface{})，由统一的响应中间件将其转换为对应的输出（JSON，XML）

## Hello, World!

下面我们来演示如何使用`elton`返回`Hello, World!`，并且添加了一些常用的中间件。

```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	// panic处理
	e.Use(middleware.NewRecover())

	// 出错处理
	e.Use(middleware.NewDefaultError())

	// 默认的请求数据解析
	e.Use(middleware.NewDefaultBodyParser())

	// not modified 304的处理
	e.Use(middleware.NewDefaultFresh())
	e.Use(middleware.NewDefaultETag())

	// 响应数据转换为json
	e.Use(middleware.NewDefaultResponder())

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
go run main.go
```

之后在浏览器中打开`http://localhost:3000/`则能看到返回的`Hello, World!`。

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

elton的路由使用[chi](https://github.com/go-chi/chi)的路由简化而来，下面是两个简单的示例。

```go
// 带参数路由
e.GET("/users/{type}", func(c *elton.Context) error {
	c.BodyBuffer = bytes.NewBufferString(c.Param("type"))
	return nil
})

// 复合参数
e.GET("/books/{category:[a-z-]+}-{type}", func(c *elton.Context) error {
	c.BodyBuffer = bytes.NewBufferString(c.Param("category") + c.Param("type"))
	return nil
})

// 带中间件的路由配置
e.GET("/users/me", func(c *elton.Context) error {
	c.Set("account", "tree.xie")
	return c.Next()
}, func(c *elton.Context) error {
	c.BodyBuffer = bytes.NewBufferString(c.GetString("account"))
	return nil
})
```

## 中间件

简单方便的中间件机制，依赖各类定制的中间件，通过各类中间件的组合，方便快捷实现各类HTTP服务，简单介绍数据响应与出错处理的中间件。

### responder

HTTP请求响应数据时，需要将数据转换为Buffer返回，而在应用时响应数据一般为各类的struct或map等结构化数据，因此elton提供了Body(interface{})字段来保存这些数据，再使用自定义的中间件将数据转换为对应的字节数据，`elton-responder`提供了将struct(map)转换为json字节并设置对应的Content-Type，对于string([]byte)则直接输出。

```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
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
			c.GetString("account"),
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
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
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

```
goos: darwin
goarch: amd64
pkg: github.com/vicanso/elton
BenchmarkRoutes-8                        5569852               214 ns/op             152 B/op          3 allocs/op
BenchmarkGetFunctionName-8              124616256                9.61 ns/op            0 B/op          0 allocs/op
BenchmarkContextGet-8                   13823508                84.1 ns/op            16 B/op          1 allocs/op
BenchmarkContextNewMap-8                157319816                7.60 ns/op            0 B/op          0 allocs/op
BenchmarkConvertServerTiming-8           1339969              1059 ns/op             360 B/op         11 allocs/op
BenchmarkGetStatus-8                    1000000000               0.282 ns/op           0 B/op          0 allocs/op
BenchmarkStatic-8                          20527             56061 ns/op           25975 B/op        628 allocs/op
BenchmarkGitHubAPI-8                       13024             92207 ns/op           33829 B/op        812 allocs/op
BenchmarkGplusAPI-8                       260158              4349 ns/op            2147 B/op         52 allocs/op
BenchmarkParseAPI-8                       135792              8433 ns/op            4288 B/op        104 allocs/op
BenchmarkRWMutexSignedKeys-8            75854030                15.9 ns/op             0 B/op          0 allocs/op
BenchmarkAtomicSignedKeys-8             833694663                1.43 ns/op            0 B/op          0 allocs/op
PASS
ok      github.com/vicanso/elton        19.403s
goos: darwin
goarch: amd64
pkg: github.com/vicanso/elton/middleware
BenchmarkGenETag-8        244416              4521 ns/op             280 B/op          7 allocs/op
BenchmarkMd5-8            196239              6011 ns/op             232 B/op          7 allocs/op
BenchmarkProxy-8           14181             76270 ns/op           15840 B/op        105 allocs/op
PASS
ok      github.com/vicanso/elton/middleware     5.429s
```
