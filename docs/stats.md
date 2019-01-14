# stats

stats中间件，包括接口响应、状态码等统计信息。

StatsInfo字段如下：

- `CID` context id
- `IP` 客户端IP
- `Method` http请求Method
- `Route` 该请求对应的路由
- `URI` 请求地址
- `Status` http请求响应状态码
- `Type` 响应状态码类型， status / 100
- `Size` 响应字节
- `Connecting` 当前正处理请求数


```go
d := cod.New()

d.Use(middleware.NewStats(middleware.StatsConfig{
  OnStats: func(info *StatsInfo, _ *cod.Context) {
    fmt.Println(info)
  },
}))

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.GET("/ping", func(c *cod.Context) (err error) {
  c.Body = "pong"
  return
})
```