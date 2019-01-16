# tracker

tracker中间件，主要用于提交类的请求，记录请求的Query、Params以及Form参数，并可以指定对哪些字段使用***的处理，以及接口的处理结果（成功或失败）。

```go
loginTracker := func(info *middleware.TrackerInfo, _ *cod.Context) {
  // 输出track日志，在实际使用中可以记录至数据库等
  fmt.Println("login:", info)
}
d.POST("/users/login", middleware.NewTracker(middleware.TrackerConfig{
  OnTrack: loginTracker,
  // 指定哪些字段做***处理
  Mask: regexp.MustCompile(`password`),
}), func(c *cod.Context) (err error) {
  c.Body = &struct {
    Account string `json:"account"`
  }{
    "tree.xie",
  }
  return
})
```