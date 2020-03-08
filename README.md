# elton 

[![Build Status](https://img.shields.io/travis/vicanso/elton.svg?label=linux+build)](https://travis-ci.org/vicanso/elton)


Elton的实现参考了[koa](https://github.com/koajs/koa)以及[echo](https://github.com/labstack/echo)，统一中间件的形式，方便定制各类中间件，所有中间件的处理方式都非常简单，如果需要转给下一中间件，则调用`Context.Next()`，如果当前中间件出错，则返回`Error`结束调用，如果无需要转至下一中间件，则无需要调用`Context.Next()`。
对于成功返回只需将响应数据赋值`Context.Body = 响应数据`，由响应中间件将Body转换为相应的响应数据，如JSON等。

```golang
package main

import (
	"log"
	"time"

	"github.com/vicanso/elton"
	errorHandler "github.com/vicanso/elton-error-handler"
	recover "github.com/vicanso/elton-recover"
	responder "github.com/vicanso/elton-responder"
	"github.com/vicanso/hes"
)

func main() {

	e := elton.New()

	// 捕捉panic异常，避免程序崩溃
	e.Use(recover.New())
	// 错误处理，将错误转换为json响应
	e.Use(errorHandler.NewDefault())
	// 请求处理时长
	e.Use(func(c *elton.Context) (err error) {
		started := time.Now()
		err = c.Next()
		log.Printf("response time:%s", time.Since(started))
		return
	})
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
			c.Get("account").(string),
			"vip",
		}
		return
	})

	e.GET("/error", func(c *elton.Context) (err error) {
		// 自定义的error
		err = &hes.Error{
			StatusCode: 400,
			Category:   "custom-error",
			Message:    "error message",
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
