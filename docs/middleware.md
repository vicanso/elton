## middleware

cod中的一些常用中间件，如`basic auth`, `body parser`等等。

在所有的中间件配置都支持`Skipper`函数，用于判断是否需要跳过此中间件，其定义如下：

```go
type (
	// Skipper check for skip middleware
	Skipper func(c *cod.Context) bool
)
```

函数返回true则表示对于此次请求，跳过此中间件。如果未设置此Skipper属性，则使用默认的函数，其实现如下：

```go
func DefaultSkipper(c *cod.Context) bool {
	return c.Committed
}
```

- [basic auth](./basic_auth.md) HTTP basic 认证，非常简单的认证方式，建议只使用于内网访问系统
- [body parser](./body_parser.md) 对提交数据转换为json格式，提交的数据类型支持：`application/json`与`application/x-www-form-urlencoded`
- [compress](./compress.md) 响应数据压缩，默认支持`gzip`，可添加更多的压缩方式，如`brotli`
- [concurrent limiter](./concurrent_limiter.md) 并发控制中间件，可以通过指定参数生成唯一的key，控制此key在特定时间内的调用，避免重复提交。主要用于一些产品购买等场景。
- [etag](./etag.md) 根据响应内容生成HTTP ETag
- [fresh](./fresh.md) 根据请求头与响应头，判断客户端缓存是否可使用(304)
- [json picker](./json_picker.md) 从响应的json数据库筛选指定的字段
- [proxy](./proxy.md) HTTP请求转发
- [recover](./recover.md) 程序出现panic异常的恢复处理
- [responder](./responder.md) 用于将c.Body(interface{})转换为相应的字节数据并且设置对应的`Content-Type`
- [stats](./stats.md) 接口统计，包括请求时长，状态码，数据长度等
- [tracker](./tracker.md) 主要用于接口参数输入，包括query，params以及form，针对一些客户提交性的操作（如购买等）可增加此中间件，便于排查。
