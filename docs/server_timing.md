---
description: HTTP Server Timing
---

elton可以非常方便的获取各中间件的处理时长，获取统计时长之后，则可方便的写入相关的统计数据或HTTP响应的Server-Timing了。

如图所示在chrome中network面板所能看得到Server-Timing展示：

![](https://raw.githubusercontent.com/vicanso/elton/master/.data/server-timing.png)


```go
package main

import (
	"github.com/vicanso/elton"
	responder "github.com/vicanso/elton-responder"
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

	fn := responder.NewDefault()
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
< Content-Type: application/json; charset=UTF-8
< Server-Timing: elton-0;dur=0;desc="entry",elton-1;dur=0.03;desc="responder",elton-2;dur=0;desc="main.main.func3"
< Date: Fri, 03 Jan 2020 13:08:50 GMT
<
* Connection #0 to host 127.0.0.1 left intact
{"name":"tree.xie","content":"Hello, World!"}
```