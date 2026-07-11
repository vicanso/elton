---
description: elton简述
---

# 前言

开始接触后端开发是从nodejs开始，最开始使用的框架是express，后来陆续接触了其它的框架，觉得最熟悉的还是koa。使用golang做后端开发时，对比使用过gin，echo以及iris三个框架，它们的用法都类似（都支持中间件，中间件的处理也类似），但是在开发过程中还是钟情于koa的处理方式，失败则throw error，成功则将响应数据赋值至ctx.body，简单易懂。

# 概述

造一个新的轮子的时候，首先考虑的是满足自己的需求，弱水三千只取一瓢饮，新轮子的满足我所需要的一瓢：无论成功还是失败的响应都应该由框架统一处理，而不是各中间件或路由处理函数直接将响应至http.ResponseWriter。为什么有这样的考虑呢？在实际开发过程中，开发人员的能力高低不一，希望可以简单的插入统一的响应处理，便于生成统计报告。具体框架主要实现以下要点：

- 请求经过中间件的处理方式为由外至内，响应时再由内至外
- 所有的处理函数都一致（参数、类型等），每个处理函数都可以是其它处理函数的前置中间件
- 请求处理成功时，直接赋值至 `Body`（`any`），由中间件将其序列化为相应的 bytes（如 json、xml 等）
- 请求处理失败时，返回error，由中间件将error转换为相应的bytes（golang中的error为interface，可自定义相应的Error实例）

elton参考koa的实现，能够简单的添加各类中间件，中间件的执行也和koa一样，如下图所示的洋葱图，从外层往内层递进，再从内层返回外层（也可以未至最内层则直接往上返回）。

<p align="center">
  <img src="https://raw.githubusercontent.com/vicanso/elton/master/.data/koa.png">
</p>


下面我们先看一下简单的处理成功与出错的例子：

```go
package main

import (
	"errors"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())
	e.Use(middleware.NewDefaultError())

	e.GET("/", func(c *elton.Context) (err error) {
		c.Body = &struct {
			Message string `json:"message,omitempty"`
		}{
			Message: "Hello world!",
		}
		return
	})

	e.GET("/error", func(c *elton.Context) (err error) {
		err = errors.New("my error")
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

如代码所示，处理过程非常简单：响应数据直接赋值至 `Body`（`any`），通过 `middleware.NewDefaultResponder()` 可将 struct 等转换为 JSON 响应（也可自定义中间件实现更多类型输出）。若处理出错，直接返回 `error`，由 `middleware.NewDefaultError()` 等错误中间件转换为对应的 HTTP 响应。此两类中间件后续会有更详细的介绍。


## 统一的HTTP响应

`elton`的HTTP响应（成功与出错）是在所有的中间件以及路由处理函数完成之后，常规处理是由框架最终将`BodyBuffer`的数据写入`http.ResponseWriter`，所有中间件与处理函数均不直接将数据写入`http.ResponseWriter`。

对于成功响应数据，为了方便开发，`elton` 提供 `ctx.Body`（类型为 `any`）保存各类响应数据，再由响应中间件转换为对应的 Buffer（如 `json.Marshal` 等）；也支持直接写入 `ResponseWriter`，但不建议使用。

处理出错都是直接返回`error`，通过自定义的error handler中间件，根据应用场景将error转换为相应的数据类型（如json）。由于统一的出错处理，因此可以在自定义的错误处理中间件极为方便的将各类出错信息汇总、统计，针对非自定义的出错（如开发不规范或一些未知出错）汇总，方便后续针对相关流程优化调整。

将HTTP响应统一处理之后，响应数据就分为三部分：状态码（int）、响应头（http.Header）、响应体（*bytes.Buffer)，就可以很方便的实现以下一些功能：

- 基于响应头的`Content-Type`以及响应体大小来判断是否对数据压缩，以及`Accept-Encoding`选择合适的压缩算法
- 基于响应体生成`ETag`以及`304`的处理
- 判断`Cache-Control`是否可缓存将GET、HEAD的响应数据直接缓存至内存或数据库中，实现URL缓存功能

为什么`elton`不建议使用直接将数据写入`ResponseWriter`的响应形式？

考虑以下场景，增加`gzip`的压缩中间件，需要对响应数据做压缩处理。如果使用直接写入数据的形式，则只能包装一层ResponseWriter，使用自定义的Writer在接收到数据时，先压缩再传递给原来的ResponseWriter，通过这样的形式可以实现数据压缩，但无法实现个性化的数据压缩，如：根据响应数据类型、响应数据长度选择不同的压缩处理。

再考虑`304`的处理场景，需要对当前响应数据计算其`ETag`再判断是否有更新，做此处理只能先将响应数据转换为字节再计算，如果直接写入`ResponseWriter`就无法实现此中间件。


## 中间件

elton的各类中间件才是真正精髓，处理函数是`Handler func(*Context) error`，可以通过Use方法添加至全局的中间件，也可单独添加至单一组或单一的路由处理。中间件处理也非常简单，如果出错，返回Error（后续的处理函数不再执行）。在当前函数中已完成处理，则无需要调用`Context.Next()`，需要转至下一处理函数，则调用`Context.Next()`则可，下面主要讲解常用的中间件实现。

```go
package main

import (
	"bytes"
	"log"
	"time"

	"github.com/vicanso/elton/v2"
)

func main() {
	e := elton.New()

	// logger
	e.Use(func(c *elton.Context) (err error) {
		err = c.Next()
		rt := c.GetHeader("X-Response-Time")
		log.Printf("%s %s - %s\n", c.Request.Method, c.Request.RequestURI, rt)
		return
	})

	// x-response-time
	e.Use(func(c *elton.Context) (err error) {
		start := time.Now()
		err = c.Next()
		c.SetHeader("X-Response-Time", time.Since(start).String())
		return
	})

	e.GET("/", func(c *elton.Context) (err error) {
		c.BodyBuffer = bytes.NewBufferString("Hello, World!")
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

### responder中间件

HTTP 的响应主要分三部分：状态码、响应头、响应体。前两部分格式统一，响应体则因应用而异。elton 最终将 `BodyBuffer` 作为 HTTP 响应体输出；应用侧可把业务数据赋给 `Body`（`any`），再由中间件转为 `BodyBuffer` 并设置 `Content-Type`（JSON / XML / 自定义均可）。

实际接口多以 JSON 为主，内置 `middleware.NewDefaultResponder()`（`NewResponder`）会将 `Body` 转为 JSON 的 `BodyBuffer`，核心处理逻辑如下：

```go
// NewResponder create a responder
func NewResponder(config ResponderConfig) elton.Handler {
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	marshal := config.Marshal
	// 如果未定义marshal
	if marshal == nil {
		marshal = json.Marshal
	}
	contentType := config.ContentType
	if contentType == "" {
		contentType = elton.MIMEApplicationJSON
	}

	return func(c *elton.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		err = c.Next()
		if err != nil {
			return
		}
		// 如果已设置了BodyBuffer，则已生成好响应数据，跳过
		if c.BodyBuffer != nil {
			return
		}

		if c.StatusCode == 0 && c.Body == nil {
			// 如果status code 与 body 都为空，则为非法响应
			err = ErrInvalidResponse
			return
		}
		// 如果body是reader，则跳过
		if c.IsReaderBody() {
			return
		}

		hadContentType := false
		// 判断是否已设置响应头的Content-Type
		if c.GetHeader(elton.HeaderContentType) != "" {
			hadContentType = true
		}

		var body []byte
		if c.Body != nil {
			switch data := c.Body.(type) {
			case string:
				if !hadContentType {
					c.SetHeader(elton.HeaderContentType, elton.MIMETextPlain)
				}
				body = []byte(data)
			case []byte:
				if !hadContentType {
					c.SetHeader(elton.HeaderContentType, elton.MIMEBinary)
				}
				body = data
			default:
				// 使用marshal转换（默认为转换为json）
				buf, e := marshal(data)
				if e != nil {
					// hes.Wrap 对非 hes 错误默认 500 + Exception
					return hes.Wrap(e, hes.WithCategory(ErrResponderCategory))
				}
				if !hadContentType {
					c.SetHeader(elton.HeaderContentType, contentType)
				}
				body = buf
			}
		}

		statusCode := c.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		if len(body) != 0 {
			c.BodyBuffer = bytes.NewBuffer(body)
		}
		c.StatusCode = statusCode
		return nil
	}
}
```

代码的处理步骤如下：

1、前置判断是否跳过中间件，主要判断条件为：是否出错，或者已设置`BodyBuffer`(表示已完成响应数据的处理)或者Body为Reader(以流的形式输出响应数据)。

2、如果Body的类型为string，则将string转换为bytes，如果未设置数据类型，则设置为`text/plain; charset=utf-8`

3、如果Body的类型为[]byte，如果未设置数据类型，则设置为`application/octet-stream`

4、对于其它类型，则使用marshal(默认为json.Marshal)转换为对应的[]byte，如果未设置数据类型，则设置Content-Type(默认为application/json; charset=utf-8)

通过此中间件，在开发时可以简单的将各种struct对象，map对象以`json`的形式返回，无需要单独处理数据转换，方便快捷。如果应用需要以xml等其它形式返回，则可自定义marshal与contentType。

### error handler中间件

elton 框架内置的默认 Error 处理只输出 `err.Error()`，状态码多为 `StatusInternalServerError`，难以满足定制场景。建议使用中间件统一处理错误；内置的 `middleware.NewDefaultError()` / `NewError` 会识别 [hes.Error](https://github.com/vicanso/hes)（含 `fmt.Errorf("%w")` 包装），生成对应状态码与 JSON/文本响应：

```go
// NewError create a error handler
func NewError(config ErrorConfig) elton.Handler {
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		err := c.Next()
		// 如果没有出错，直接返回
		if err == nil {
			return nil
		}
		he, ok := hes.As(err)
		if !ok {
			// 非hes的error包装为500出错异常
			he = hes.Wrap(err, hes.WithCategory(ErrErrorCategory))
		}
		c.StatusCode = he.StatusOrInternal()
		if config.ResponseType == "json" ||
			strings.Contains(c.GetRequestHeader("Accept"), "application/json") {
			// 序列化失败时降级为 text 输出
			if buf, e := he.ToJSON(); e == nil {
				c.BodyBuffer = bytes.NewBuffer(buf)
				c.SetHeader(elton.HeaderContentType, elton.MIMEApplicationJSON)
				return nil
			}
		}
		c.BodyBuffer = bytes.NewBufferString(he.Error())
		c.SetHeader(elton.HeaderContentType, elton.MIMETextPlain)

		return nil
	}
}
```

# 后记

Elton提供更简单方便的WEB开发体验，实现的代码非常简单，更多的功能都依赖于各类中间件。需要查阅更多的中间件以及文档说明请查阅中间件中列表中的各类中间件。