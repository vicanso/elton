---
description: 路由参数校验
---

在路由配置中，会有各类的路由参数，Elton提供简单的校验函数，建议对每个参数都添加相应的校验参数，增强系统安全性。

校验函数非常简单，`Validator func(value string) error`，它的参数是字符串（url中都只能是字符串)，如果参数不符合，则返回Error，如果符合，则返回nil。需要注意，路由参数校验的函数是针对Elton实例的，因此对路由参数名配置必须避免冲突。如下面示例中，书籍的ID则定义为`bookID`：

```go
package main

import (
	"regexp"

	"github.com/vicanso/elton"
	responder "github.com/vicanso/elton-responder"
	"github.com/vicanso/hes"
)

func main() {
	e := elton.New()

	e.AddValidator("bookID", func(value string) error {
		// book id 必须为数字
		r := regexp.MustCompile(`^[1-9]\d{0,3}$`)
		if !r.MatchString(value) {
			return hes.New("book id shoule be number lt 10000")
		}
		return nil
	})

	e.Use(responder.NewDefault())
	e.GET("/:bookID", func(c *elton.Context) (err error) {
		c.Body = &struct {
			Name string `json:"name,omitempty"`
		}{
			"代码大全",
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
curl -v 'http://127.0.0.1:3000/abcd'
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to 127.0.0.1 (127.0.0.1) port 3000 (#0)
> GET /abcd HTTP/1.1
> Host: 127.0.0.1:3000
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 400 Bad Request
< Date: Fri, 03 Jan 2020 12:00:43 GMT
< Content-Length: 41
< Content-Type: text/plain; charset=utf-8
<
* Connection #0 to host 127.0.0.1 left intact
message=book id shoule be number lt 10000
```