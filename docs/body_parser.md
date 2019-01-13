# body parser

body parser中间件，用于将提交的数据转换为json数据（字节），支持的的提交数据格式有两种`application/json`与`application/x-www-form-urlencoded`。此中间件只针对`post`, `patch`以及`put`三种类型的请求。


```go
d := cod.New()

// 用于将c.Body(interface{})转换为字节
d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.Use(middleware.NewBodyParser(middleware.BodyParserConfig{
  // 10KB，限制数据长度，如果不配置，则默认为50KB
  Limit:                1024 * 10,
  // 是否忽略json的数据类型
  IgnoreJSON:           false,
  // 是否忽略form url encoded的数据类型
  IgnoreFormURLEncoded: true,
}))

d.POST("/login", func(c *cod.Context) (err error) {
  c.Body = &struct {
    Name     string `json:"name,omitempty"`
    PostBody string `json:"postBody,omitempty"`
  }{
    "tree.xie",
    string(c.RequestBody),
  }
  return
})
```