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

### IgnoreNext

设置是否忽略后续的Handler，如果设置为true，则后续的所有Handler都不再执行，包括全局中间件或者单独的路由处理函数。

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

### BodyBytes

HTTP的响应数据（字节），此数据为真正返回的响应体，responder中间件就是将Body转换为字节(BodyBytes)，并写入相应的`Content-Type`。

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

### Cookie

获取HTTP请求头中的cookie。

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

### SetCookie

设置Cookie至HTTP响应头中。

```go
req := httptest.NewRequest("GET", "/users/me", nil)
resp := httptest.NewRecorder()
c := cod.NewContext(resp, req)
c.SetCookie(&http.Cookie{
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
