# responder

responder中间件，此中间件主要做以下的处理：

- 如果Handler的执行返回Error，则将Error转换为相应的HTTP响应数据（响应状态码与响应体）
- 将响应的数据Body(interface{})根据其类型转换为[]byte，并设置`Content-Type`
- 设置响应状态码与响应体(BodyBuffer)

此中间件在生成响应数据前，会先判断c.BodyBuffer是否为nil，如果不为nil，则表示已生成响应数据，此中间件无需处理。

```go
d := cod.New()

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.GET("/users/me", func(c *cod.Context) (err error) {
  c.Body = &struct {
    Name string `json:"name"`
  }{
    "tree.xie",
  }
  return
})
```