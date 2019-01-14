# proxy

proxy转发中间件，用于将当前请求转发至其它服务，此中间件的参数如下：


- `Target` 转发的服务地址，如果未设置此属性，则使用TargetPicker函数获取对应的转发地址，两者必须设置其中一个。

- `TargetPicker` 转发服务地址选择函数，如果需要动态返回相应的转发地址（如多个IP中选择一个），则可自定义相应的选择函数。

- `Rewrites` url重写的配置列表，如`"/proxy/*:/$1"`表示将`/proxy`删除后再转发。

- `Host` 转发时需要设置的Host（有些反向代理依赖于HTTP请求中的Host转发）

- `Transport` http.Transport，可以自定义转发使用的Transport

```go
target, _ := url.Parse("https://aslant.site/")
proxyMid := middleware.NewProxy(middleware.ProxyConfig{
  Target: target,
  Host:   "aslant.site",
  // 转发时重写url
  Rewrites: []string{
    "/proxy/*:/$1",
  },
  Transport: &http.Transport{
    // 禁用 keep alive
    DisableKeepAlives: true,
  },
})
d.GET("/proxy/*path", proxyMid, func(c *cod.Context) error {
  return nil
})
```
