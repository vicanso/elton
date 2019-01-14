# recover

recover中间件，在程序出现异常时的处理，在生产环境中建议参考此中间件编写，而非直接使用。因为此类异常一般都是程序BUG，需要特别增加一些告警之类的处理（如邮件提醒等）。

```go
d := cod.New()

d.Use(middleware.NewRecover())

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.GET("/ping", func(c *cod.Context) (err error) {
  c.Body = "pong"
  return
})
```