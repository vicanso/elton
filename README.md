# elton 

[![Build Status](https://img.shields.io/travis/vicanso/elton.svg?label=linux+build)](https://travis-ci.org/vicanso/elton)


Elton的实现参考了[koa](https://github.com/koajs/koa)以及[echo](https://github.com/labstack/echo)，统一中间件的形式，方便定制各类中间件，所有中间件的处理方式都非常简单，如果需要转给下一中间件，则调用`Context.Next()`，如果当前中间件出错，则返回`Error`结束调用，如果无需要转至下一中间件，则无需要调用`Context.Next()`。
对于成功返回只需将响应数据赋值`Context.Body = 响应数据`，由响应中间件将Body转换为相应的响应数据，如JSON等。


## Hello, World!

下面我们来演示如何使用`elton`返回`Hello, World!`。

```go
package main

import (
	"github.com/vicanso/elton"
)

func main() {
    e := elton.New()

    e.GET("/", func(c *elton.Context) error {
        c.BodyBuffer = bytes.NewBufferString("Hello, World!")
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

elton与gin一样使用httprouter来处理路由，而每个路由可以添加多个中间件处理函数，根据路由与及HTTP请求方法指定不同的路由处理函数。而全局的中间件则可通过`Use`方法来添加。

```go
e.Use(...func(*elton.Context) error)
e.Method(path string, ...func(*elton.Context) error)
```

- `e` 为`elton`实例化对象
- `Method` 为HTTP的请求方法，如：`GET`, `PUT`, `POST`等等
- `path` 为HTTP路由路径
- `func(*elton.Context) error` 为路由处理函数（中间件），当匹配的路由被请求时，对应的处理函数则会被调用

### 路由示例

elton的路由使用[httprouter](https://github.com/julienschmidt/httprouter)，下面是两个简单的示例，更多的使用方式可以参考httprouter。

```go
// 带参数的路由配置
e.GET("/books/:type", func(c *elton.Context) error {
    c.BodyBuffer = bytes.NewBufferString(c.Param("type"))
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

### responser

HTTP请求响应数据时，需要将数据转换为Buffer返回，而在应用时响应数据一般为各类的struct或map等结构化数据，因此elton提供了Body(interface{})字段来保存这些数据，再使用自定义的中间件将数据转换为对应的字节数据，[elton-responder](https://github.com/vicanso/elton-responder)提供了转数据转换为json字节并设置对应的Content-Type。

```go
package main

import (
	"github.com/vicanso/elton"
	responder "github.com/vicanso/elton-responder"
)

func main() {

	e := elton.New()
	// 对响应数据 c.Body 转换为相应的json响应
	e.Use(responder.NewDefault())

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

### error-handler

当请求处理失败时，直接返回error则可，elton从error中获取出错信息并输出。默认的出错处理并不适合实际应用场景，建议使用自定义出错类配合中间件，便于统一的错误处理，程序监控。

```go
package main

import (
	"github.com/vicanso/elton"
	errorhandler "github.com/vicanso/elton-error-handler"
	"github.com/vicanso/hes"
)

func main() {

	e := elton.New()
	// 指定出错以json的形式返回
	e.Use(errorhandler.New(errorhandler.Config{
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

## bench

```
BenchmarkRoutes-8                	 3489271	       343 ns/op	     376 B/op	       4 allocs/op
BenchmarkGetFunctionName-8       	129711422	         9.23 ns/op	       0 B/op	       0 allocs/op
BenchmarkContextGet-8            	14131228	        84.4 ns/op	      16 B/op	       1 allocs/op
BenchmarkContextNewMap-8         	183387170	         6.52 ns/op	       0 B/op	       0 allocs/op
BenchmarkConvertServerTiming-8   	 1430475	       839 ns/op	     360 B/op	      11 allocs/op
BenchmarkGetStatus-8             	1000000000	         0.272 ns/op	       0 B/op	       0 allocs/op
BenchmarkRWMutexSignedKeys-8     	35028435	        33.5 ns/op
BenchmarkAtomicSignedKeys-8      	602747588	         1.99 ns/op
```
