# Elton 

[![license](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/vicanso/elton/blob/master/LICENSE)
[![Build Status](https://github.com/vicanso/elton/workflows/Test/badge.svg)](https://github.com/vicanso/elton/actions)

![Alt](https://repobeats.axiom.co/api/embed/4f64b99db39c6a75b6980ebb3c756244b246a718.svg "Repobeats analytics image")

Elton的实现参考了[koa](https://github.com/koajs/koa)以及[echo](https://github.com/labstack/echo)，中间件的调用为洋葱模型：请求由外至内，响应由内至外。主要特性如下：

- 处理函数（中间件）均以返回error的形式响应出错，方便使用统一的出错处理中间件将出错统一转换为对应的输出（JSON），并根据出错的类型等生成各类统计分析
- 成功响应数据直接赋值至Context.Body（interface{})，由统一的响应中间件将其转换为对应的输出（JSON，XML）

如何使用`elton`开发WEB后端程序，可以参考[一步一步学习如何使用elton](https://treexie.gitbook.io/elton-beginner/)

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

简单方便的中间件机制，依赖各类定制的中间件，通过各类中间件的组合，方便快捷实现各类HTTP服务，简单介绍数据响应与出错处理的中间件。需要注意，elton中默认不会执行所有的中间件，每个中间件决定是否需要执行后续处理，如果需要则调用Next()函数，与gin不一样(gin默认为执行所有，若不希望执行后续的中间件，则调用Abort)。

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
BenchmarkRoutes-8                        6925746               169.4 ns/op           120 B/op          2 allocs/op
BenchmarkGetFunctionName-8              136577900                9.265 ns/op           0 B/op          0 allocs/op
BenchmarkContextGet-8                   15311328                78.11 ns/op           16 B/op          1 allocs/op
BenchmarkContextNewMap-8                187684261                6.276 ns/op           0 B/op          0 allocs/op
BenchmarkConvertServerTiming-8           1484379               835.8 ns/op           360 B/op         11 allocs/op
BenchmarkGetStatus-8                    1000000000               0.2817 ns/op          0 B/op          0 allocs/op
BenchmarkFresh-8                          955664              1233 ns/op             416 B/op         10 allocs/op
BenchmarkStatic-8                          25128             46709 ns/op           20794 B/op        471 allocs/op
BenchmarkGitHubAPI-8                       14724             76190 ns/op           27175 B/op        609 allocs/op
BenchmarkGplusAPI-8                       326769              3659 ns/op            1717 B/op         39 allocs/op
BenchmarkParseAPI-8                       162340              6989 ns/op            3435 B/op         78 allocs/op
BenchmarkRWMutexSignedKeys-8            71757390                17.51 ns/op            0 B/op          0 allocs/op
BenchmarkAtomicSignedKeys-8             923771157                1.297 ns/op           0 B/op          0 allocs/op
PASS
ok      github.com/vicanso/elton        20.225s
goos: darwin
goarch: amd64
pkg: github.com/vicanso/elton/middleware
BenchmarkGenETag-8                        230718              4409 ns/op             160 B/op          6 allocs/op
BenchmarkMd5-8                            200134              5958 ns/op             120 B/op          6 allocs/op
BenchmarkNewShortHTTPHeader-8           10220961               116.4 ns/op            80 B/op          2 allocs/op
BenchmarkNewHTTPHeader-8                 4368654               277.1 ns/op            88 B/op          3 allocs/op
BenchmarkNewHTTPHeaders-8                 384062              2822 ns/op            1182 B/op         23 allocs/op
BenchmarkHTTPHeaderMarshal-8              225123              4664 ns/op            1344 B/op         21 allocs/op
BenchmarkToHTTPHeader-8                   296210              3834 ns/op            1272 B/op         34 allocs/op
BenchmarkHTTPHeaderUnmarshal-8            120136             10108 ns/op            1888 B/op         50 allocs/op
BenchmarkProxy-8                           13393             85170 ns/op           16031 B/op        104 allocs/op
PASS
ok      github.com/vicanso/elton/middleware     14.007s
```
