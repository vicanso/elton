---
description: HTTP Server Timing
---

elton 可记录各中间件耗时，并写入统计或 HTTP `Server-Timing`。开启方式：`e.EnableTrace = true`，并注册 `e.OnTrace`；中间件建议 `UseWithName` / `SetFunctionName` 命名。

注意：`TraceFromContext` / `c.Trace()` 在 context 尚无 trace 时会返回**未挂载**的新对象，不会进入 `OnTrace`；依赖框架统计时请开启 `EnableTrace`，或业务侧使用 `c.NewTrace()`。详见 [application.md EnableTrace](./application.md#enabletrace)。

如图所示在 Chrome Network 面板中的 Server-Timing：

![](https://raw.githubusercontent.com/vicanso/elton/master/.data/server-timing.png)


```go
package main

import (
	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {

	e := elton.New()
	e.EnableTrace = true
	e.OnTrace(func(c *elton.Context, traceInfos elton.TraceInfos) {
		serverTiming := string(traceInfos.ServerTiming("elton-"))
		c.SetHeader(elton.HeaderServerTiming, serverTiming)
	})

	entry := func(c *elton.Context) (err error) {
		c.ID = "random id"
		c.NoCache()
		return c.Next()
	}
	e.Use(entry)
	// 设置中间件的名称，若不设置从runtime中获取
	// 对于公共的中间件，建议指定名称
	e.SetFunctionName(entry, "entry")

	fn := middleware.NewDefaultResponder()
	e.Use(fn)
	e.SetFunctionName(fn, "responder")

	e.GET("/", func(c *elton.Context) (err error) {
		c.Body = &struct {
			Name    string `json:"name,omitempty"`
			Content string `json:"content,omitempty"`
		}{
			"tree.xie",
			"Hello, World!",
		}
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

```
curl -v 'http://127.0.0.1:3000/'
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to 127.0.0.1 (127.0.0.1) port 3000 (#0)
> GET / HTTP/1.1
> Host: 127.0.0.1:3000
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Cache-Control: no-cache
< Content-Length: 44
< Content-Type: application/json; charset=utf-8
< Server-Timing: elton-0;dur=0;desc="entry",elton-1;dur=0.03;desc="responder",elton-2;dur=0;desc="main.main.func3"
< Date: Fri, 03 Jan 2020 13:08:50 GMT
<
* Connection #0 to host 127.0.0.1 left intact
{"name":"tree.xie","content":"Hello, World!"}
```