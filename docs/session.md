# session

session中间件。

```go
client := ss.NewRedisClient(&redis.Options{
  Addr: "localhost:6379",
})

createStore := func(c *cod.Context) middleware.Store {
  rs := ss.NewRedisStore(client)
  rs.SetOptions(&ss.Options{
    TTL: 30 * time.Second,
    Key: "jt",
    // 用于将id与密钥生成校验串，建议配置此参数，并注意保密
    SignKeys: []string{
      "secret1",
      "secret2",
    },
    CookieOptions: &cookies.Options{
      HttpOnly: true,
      Path:     "/",
    },
    // 默认的id生成函数并不能保证大量用户时唯一性，建议使用uuid等之类的生成方式
    IDGenerator: func() string {
      t := time.Now()
      entropy := rand.New(rand.NewSource(t.UnixNano()))
      return ulid.MustNew(ulid.Timestamp(t), entropy).String()
    },
  })
  // 设置context
  rs.SetContext(c)
  return rs
}

d := cod.New()

d.Use(middleware.NewRecover())

d.Use(middleware.NewFresh(middleware.FreshConfig{}))
d.Use(middleware.NewETag(middleware.ETagConfig{}))

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

// session中间件
sessionMid := middleware.NewSession(middleware.SessionConfig{
  CreateStore: createStore,
})
// 建议按根据增加session中间件，而不是全局Use
d.GET("/users/me", sessionMid, func(c *cod.Context) (err error) {
  se := c.Get(cod.SessionKey).(*middleware.Session)
  count := se.GetInt("count")
  // session中的count每次+1
  se.Set("count", count+1)
  c.Body = &struct {
    Account string `json:"account"`
    Count   int    `json:"count"`
  }{
    "tree.xie",
    count,
  }
  return
})

d.ListenAndServe(":8001")
```