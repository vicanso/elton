## Cod

实现HTTP服务的监听、中间件的顺序调用以及路由的选择调用。


## New

创建一个Cod的实例，并初始化相应的http.Server。

```go
d := cod.New()
```

## NewWithoutServer

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
d.OnTrace(func(c *cod.Context, traceInfos []*cod.TraceInfo) {
	buf, _ := json.Marshal(traceInfos)
	// [{"name":"responder","duration":21755},{"name":"main.main.func2","duration":1750}]
	log.Println(string(buf))
	// cod-0;dur=0.021755;desc="responder",cod-1;dur=0.00175;desc="main.main.func2"
	log.Println(cod.ConvertToServerTiming(traceInfos, "cod-"))
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
