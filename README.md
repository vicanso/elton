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


	// 请求处理时长
	d.Use(func(c *cod.Context) (err error) {
		started := time.Now()
		err = c.Next()
		log.Printf("response time:%s", time.Since(started))
		return
	})

	// 针对出错error生成相应的HTTP响应数据（http状态码以及响应数据）
	d.Use(middleware.NewDefaultErrorHandler()))

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

	d.GET("/error", func(c *cod.Context) (err error) {
		err = errors.New("abcd")
		return
	})

	d.ListenAndServe(":8001")
}
```

上面的例子已经实现了简单的HTTP响应（得益于golang自带http的强大），整个框架中主要有两个struct：Cod与Context，下面我们来详细介绍这两个struct。

## Cod

实现HTTP服务的监听、中间件的顺序调用以及路由的选择调用。

创建一个Cod的实例，并初始化相应的http.Server。

```go
d := cod.New()
```

创建一个Cod的实例，并未初始化相应的http.Server，可根据需要再初始化。

```go
d := cod.NewWithoutServer()
s := &http.Server{
	Handler: d,
}
d.Server = s
```

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
  return hes.New("abcd")
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
  return hes.New("abcd")
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

### EnableTrace

是否启用调用跟踪，设置此参数为true，则会记录每个Handler的调用时长（前一个Handler包含后面Handler的处理时长）。

```go
d := cod.New()

d.EnableTrace = true
d.OnTrace(func(c *cod.Context, traceInfos []*cod.TraceInfo) {
	log.Println(traceInfos[0])
})

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.GET("/ping", func(c *cod.Context) (err error) {
	c.Body = "pong"
	return
})
d.ListenAndServe(":8001")
```

### Keys

用于生成带签名的cookie的密钥，基于[keygrip](https://github.com/vicanso/keygrip)来生成与校验是否合法。

```go
d := cod.New()
d.Keys = []string{
	"secret",
	"cuttlefish",
}
```

### SetFunctionName

设置函数名字，主要用于trace中统计时的函数展示，如果需要统计Handler的处理时间，建议指定函数名称，便于统计信息的记录。

```go
// 未设置函数名称
d := cod.New()

d.EnableTrace = true
d.OnTrace(func(c *cod.Context, traceInfos []*cod.TraceInfo) {
	buf, _ := json.Marshal(traceInfos)
	// [{"name":"github.com/vicanso/test/vendor/github.com/vicanso/cod/middleware.NewResponder.func1","duration":10488},{"name":"main.main.func2","duration":1160}]
	log.Println(string(buf))
})

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.GET("/ping", func(c *cod.Context) (err error) {
	c.Body = "pong"
	return
})
d.ListenAndServe(":8001")
```

```go
// 设置responder中间件的名称
d := cod.New()

d.EnableTrace = true
d.OnTrace(func(c *cod.Context, traceInfos cod.TraceInfos) {
	buf, _ := json.Marshal(traceInfos)
	// [{"name":"responder","duration":21755},{"name":"main.main.func2","duration":1750}]
	log.Println(string(buf))
	// cod-0;dur=0.021755;desc="responder",cod-1;dur=0.00175;desc="main.main.func2"
	log.Println(traceInfos.ServerTiming("cod-"))
})
fn := middleware.NewResponder(middleware.ResponderConfig{})
d.Use(fn)
d.SetFunctionName(fn, "responder")

d.GET("/ping", func(c *cod.Context) (err error) {
	c.Body = "pong"
	return
})
d.ListenAndServe(":8001")
```

### ListenAndServe

监听并提供HTTP服务。

```go
d := cod.New()

d.ListenAndServe(":8001")
```

### Serve

提供HTTP服务。

```go
ln, _ := net.Listen("tcp", "127.0.0.1:0")
d := cod.New()
d.Serve(ln)
```

### Close

关闭HTTP服务。

### ServeHTTP

http.Handler Interface的实现，在此函数中根据HTTP请求的Method与URL.Path，从Router(httprouter)中选择符合的Handler，若无符合的，则触发404。

### Handle

添加Handler的处理函数，配置请求的Method与Path，添加相应的处理函数，Path的相关配置与[httprouter](https://github.com/julienschmidt/httprouter)一致。

```
d := cod.New()

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

noop := func(c *cod.Context) error {
	return c.Next()
}

d.Handle("GET", "/ping", noop, func(c *cod.Context) (err error) {
	c.Body = "pong"
	return
})

d.Handle("POST", "/users/:type", func(c *cod.Context) (err error) {
	c.Body = "OK"
	return
})

d.Handle("GET", "/files/*file", func(c *cod.Context) (err error) {
	c.Body = "file content"
	return
})

d.ListenAndServe(":8001")
```

Cod还支持GET，POST，PUT，PATCH，DELETE，HEAD，TRACE以及OPTIONS的方法，这几个方法与`Handle`一致，Method则为相对应的处理，下面两个例子的处理是完全相同的。

```go
d.Handle("GET", "/ping", noop, func(c *cod.Context) (err error) {
	c.Body = "pong"
	return
})
```

```go
d.GET("/ping", noop, func(c *cod.Context) (err error) {
	c.Body = "pong"
	return
})
```

### ALL

添加8个Method的处理函数，包括GET，POST，PUT，PATCH，DELETE，HEAD，TRACE以及OPTIONS，尽量只根据路由需要，添加相应的Method，不建议直接使用此函数。

```go
d.ALL("/ping", noop, func(c *cod.Context) (err error) {
	c.Body = "pong"
	return
})
```


### Use

添加全局中间件处理函数，对于所有路由都需要使用到的中间件，则使用此函数添加，若非所有路由都使用到，可以只添加到相应的Group或者就单独添加至Handler。特别需要注意的是，如session之类需要读取数据库的，如非必要，不要使用全局中间件形式。

```go
d := cod.New()

// 记录HTTP请求的时间、响应码
d.Use(func(c *cod.Context) (err error) {
	startedAt := time.Now()
	req := c.Request
	err = c.Next()
	log.Printf("%s %s %d use %s", req.Method, req.URL.RequestURI(), c.StatusCode, time.Since(startedAt).String())
	return err
})

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.GET("/ping", func(c *cod.Context) (err error) {
	c.Body = "pong"
	return
})

d.ListenAndServe(":8001")
```

### AddGroup

将group中的所有路由处理添加至cod。

```go
d := cod.New()
userGroup := NewGroup("/users", func(c *Context) error {
	return c.Next()
})
d.AddGroup(userGroup)
```

### OnError

添加Error的监听函数，如果当任一Handler的处理返回Error，并且其它的Handler并未将此Error处理，则会触发error事件。

```go
d := cod.New()

d.OnError(func(c *cod.Context, err error) {
	// 发送邮件告警等
	log.Println("unhandle error, " + err.Error())
})

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.GET("/ping", func(c *cod.Context) (err error) {
	c.Body = "pong"
	return
})

d.ListenAndServe(":8001")
```


## Context


HTTP处理中的Context，在各Hadnler中传递的实例，它包括了HTTP请求、响应以及路由参数等。

### Request

http.Request实例，包含HTTP请求的各相关信息，相关的使用方式可直接查看官方文件。如果Context无提供相应的方法或属性时，才使用此对象。

### Response

http.ResponseWriter，用于设置HTTP响应相关状态码、响应头与响应数据，context有各类函数操作此对象，一般无需通过直接操作此对象。

### Headers

HTTP响应头，默认初始化为Response的Headers，此http.Header为响应头。

### Committed

是否已将响应数据返回（状态码、数据等已写入至Response），除非需要单独处理数据的响应，否则不要设置此属性。

### ID

Context ID，如果有设置Cod.GenerateID，则在每次接收到请求，创建Context之后，调用`GenerateID`生成，一般用于日志或者统计中唯一标识当前请求。

### Route

当前对应的路由。

### Next

next函数，此函数会在获取请求时自动生成，无需调整。

### Params

路由参数对象，它等于httprouter路由匹配生成的`httprouter.Params`。

### StatusCode

HTTP响应码，设置HTTP请求处理结果的响应码。

### Body

HTTP响应数据，此属性为interface{}，因此可以设置不同的数据类型（与koa类似）。注意：设置Body之后，还需要使用中间件`responder`来将此属性转换为字节，并设置相应的`Content-Type`，此中间件主要将各类的struct转换为json，对于具体的实现可以查阅代码，或者自己实现相应的responder。

### BodyBuffer

HTTP的响应数据缓冲（字节），此数据为真正返回的响应体，responder中间件就是将Body转换为字节(BodyBuffer)，并写入相应的`Content-Type`。

### RequestBody

HTTP请求体，对于`POST`，`PUT`以及`PATCH`提交数据的请求，此字段用于保存请求体。注意：默认cod中并未从请求中读取相应的请求体，需要使用`body_parser`中间件来生成或者自定义相应的中间件。

### Reset

重置函数，将Context的属性重置，主要用于sync.Pool中复用提升性能。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
// &{GET /users/me HTTP/1.1 1 1 map[] {} <nil> 0 [] false example.com map[] map[] <nil> map[] 192.0.2.1:1234 /users/me <nil> <nil> <nil> <nil>}
fmt.Println(c.Request)
c.Reset()
// <nil>
fmt.Println(c.Request)
```

### RemoteAddr

获取请求端的IP

```go
fmt.Println(c.RemoteAddr())
```

### RealIP

获取客户端的真实IP，先判断请求头是否有`X-Forwarded-For`，如果没有再取`X-Real-Ip`，都没有则从连接IP中取。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
req.Header.Set("X-Forwarded-For", "8.8.8.8")
c := cod.NewContext(resp, req)
// 8.8.8.8
fmt.Println(c.RealIP())
```

### Param

获取路由的参数。

```go
// curl 'http://127.0.0.1:8001/users/me'
d.GET("/users/:type", func(c *cod.Context) (err error) {
  t := c.Param("type")
  // me
  fmt.Println(t)
  c.Body = t
  return
})
```

### QueryParam

获取query的参数值，此函数返回的并非字符串数组，只取数组的第一个，如果query中的相同的key的使用，请直接使用`Request.URL.Query()`来获取。

```go
resp := httptest.NewRecorder()
req := httptest.NewRequest("GET", "/users/me?type=vip", nil)
c := cod.NewContext(resp, req)
// vip
fmt.Println(c.QueryParam("type"))
```

### Query

获取请求的querystring，此函数返回的query对象为map[string]string，不同于原有的map[string][]string，因为使用相同的key的场景不多，因此增加此函数方便使用。如果有相同的key的场景，请直接使用`Request.URL.Query()`来获取。

```go
resp := httptest.NewRecorder()
req := httptest.NewRequest("GET", "/users/me?type=vip", nil)
c := cod.NewContext(resp, req)
// map[type:vip]
fmt.Println(c.Query())
```

### Redirect

重定向当前请求。

```go
d.GET("/redirect", func(c *cod.Context) (err error) {
  c.Redirect(301, "/ping")
  return
})
```

### Set

设置临时保存的值至context，在context的生命周期内有效。

```go
d.Use(func(c *cod.Context) error {
  c.Set("id", rand.Int())
  return c.Next()
})

d.GET("/ping", func(c *cod.Context) (err error) {
  // 6129484611666145821
  fmt.Println(c.Get("id").(int))
  c.Body = "pong"
  return
})
```

### Get

从context中获取保存的值，注意返回的为interface{}类型，需要自己做类型转换。

```go
d.Use(func(c *cod.Context) error {
  c.Set("id", rand.Int())
  return c.Next()
})

d.GET("/ping", func(c *cod.Context) (err error) {
  // 6129484611666145821
  fmt.Println(c.Get("id").(int))
  c.Body = "pong"
  return
})
```

### GetRequestHeader

从HTTP请求头中获取相应的值。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
fmt.Println(c.GetRequestHeader("X-Token"))
```

### Header

返回HTTP响应头。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
c.SetHeader("X-Response-Id", "abc")
// map[X-Response-Id:[abc]]
fmt.Println(c.Header())
```

### GetHeader

从HTTP响应头中获取相应的值。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
c.SetHeader("X-Response-Id", "abc")
// abc
fmt.Println(c.GetHeader("X-Response-Id"))
```

### SetHeader

设置HTTP的响应头。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
c.SetHeader("X-Response-Id", "abc")
// abc
fmt.Println(c.GetHeader("X-Response-Id"))
```

### AddHeader

添加HTTP响应头，用于添加多组相同名字的响应头，如Set-Cookie等。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
c.AddHeader("X-Response-Id", "abc")
c.AddHeader("X-Response-Id", "def")
// map[X-Response-Id:[abc def]]
fmt.Println(c.Header())
```

### Cookie/SignedCookie

获取HTTP请求头中的cookie。SignedCookie则会根据初始化Cod时配置的Keys来校验cookie是否符合，符合才返回。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
req.AddCookie(&http.Cookie{
  Name:  "jt",
  Value: "abc",
})
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
// jt=abc <nil>
fmt.Println(c.Cookie("jt"))
```

### AddCookie/AddSignedCookie

设置Cookie至HTTP响应头中。AddSignedCookie则根据当前的Cookie以及初化cod时配置的Keys再生成一个校验cookie(Name为当前Cookie的Name + ".sig")。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
c.AddCookie(&http.Cookie{
  Name:  "jt",
  Value: "abc",
})
// map[Set-Cookie:[jt=abc]]
fmt.Println(c.Header())
```

### NoContent

设置HTTP请求的响应状态码为204，响应体为空。

```go
d.GET("/no-content", func(c *cod.Context) (err error) {
  c.NoContent()
  return
})
```

### NotModified

设置HTTP请求的响应状态码为304，响应体为空。注意此方法判断是否客户端的缓存数据与服务端的响应数据一致再使用，不建议自己调用此函数，建议使用中间件`fresh`。

```go
d.GET("/not-modified", func(c *cod.Context) (err error) {
  c.NotModified()
  return
})
```

### Created

设置HTTP请求的响应码为201，并设置body。

```go
d.POST("/users", func(c *cod.Context) (err error) {
  c.Created(map[string]string{
    "account": "tree.xie",
  })
  return
})
```

### NoCache

设置HTTP响应头的`Cache-Control: no-cache`。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
c.NoCache()
// map[Cache-Control:[no-cache]]
fmt.Println(c.Header())
```

### NoStore

设置HTTP响应头的`Cache-Control: no-store`。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
c.NoCache()
// map[Cache-Control:[no-store]]
fmt.Println(c.Header())
```

### CacheMaxAge

设置HTTP响应头的`Cache-Control: public, max-age=x`。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
c.CacheMaxAge("1m")
// map[Cache-Control:[public, max-age=60]]
fmt.Println(c.Header())
```

### SetContentTypeByExt

通过文件（文件后缀）设置Content-Type。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
c.SetContentTypeByExt("user.json")
// map[Content-Type:[application/json]]
fmt.Println(c.Header())
```

### DisableReuse

禁止context复用，如果context在所有handler执行之后，还需要使用（如设置了超时出错，但无法对正在执行的handler中断，此时context还在使用中），则需要调用此函数禁用context的复用。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
c.DisableReuse()
```

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