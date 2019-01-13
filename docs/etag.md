# ETag

根据响应数据生成ETag，使用sha1算法计算数据的hash值，最终的ETag是`"长度-hash"`的形式。

```go
d := cod.New()

d.Use(middleware.NewETag(middleware.ETagConfig{}))

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.GET("/ping", func(c *cod.Context) (err error) {
  c.Body = "pong"
  return
})
```