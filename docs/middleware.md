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

