# error handler

出错处理中间件，此中间件将出错的error转换为对应的HTTP响应数据。

```go
d := cod.New()

d.Use(middleware.NewErrorHandler(middleware.ErrorHandlerConfig{}))

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.GET("/users/me", func(c *cod.Context) (err error) {
  return errors.New("abcd")
})
```