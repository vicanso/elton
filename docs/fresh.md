# fresh

校验响应数据与客户端缓存数据是否一致(304)，此中间件需要在ETag之前调用。

```go
d := cod.New()

d.Use(middleware.NewFresh(middleware.FreshConfig{}))
d.Use(middleware.NewETag(middleware.ETagConfig{}))

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.GET("/ping", func(c *cod.Context) (err error) {
  c.Body = "pong"
  return
})
```