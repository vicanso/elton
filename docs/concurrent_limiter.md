# concurrent limiter

并发限制中间件，主要用于一些并发限制的场景，避免客户多次提交，如用户注册，在线支付等。

## ConcurrentLimiterConfig

中间件参数配置，包括`Keys`与`Lock`函数，用于指定如何生成该请求对应的key以及lock的处理。下面看如下的配置：

```go
ConcurrentLimiterConfig{
  Keys: []string{
    ":ip",
    "h:X-Token",
    "q:type",
    "p:id",
    "account",
  },
  Lock: func(key string, c *cod.Context) (success bool, unlock func(), err error) {
    // 具体代码略
    return
  }
}
```

其中第一个`:ip`表示客户端的IP，每二个`h:X-Token`表示从请求头中获取`X-Token`的值，第三个`q:type`表示从`querystring`中获取`type`的值，第四个`p:id`表示从路由参数`params`中获取`id`的值，最后一个则表示从提交数据中获取`account`，将获取到的值以`,`拼接，则是该请求对应的`key`。

Lock函数返回对该key的锁是否可以（建议使用redis等实现锁），如下代码所示，对相同IP的相同账号的登录每10秒只允许一次：


```go
lock := func(key string, _ *cod.Context) (success bool, unlock func(), err error) {
  // 增加key的前缀，避免有冲突
  k := "concurrent-limiter-login," + key
  // 使用redis的 set not exists，并设置ttl为10
  ttl := 10 * time.Second
  done, err := redisClient.SetNX(k, true, ttl).Result()
  if err != nil || !done{
    return
  }
  success = true
  // 如果设置了unlock函数，在中间件完成时，会删除锁
  // 如果不希望直接删除，希望ttl过期自动删除，则不需要赋值此函数
  unlock = func()  {
		redisClient.Del(key).Result()
  }
  return
}
```
