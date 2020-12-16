---
description: Context的相关方法说明
---

# Context

## Request

http.Request实例，包含HTTP请求的各相关信息，相关的使用方式可直接查看官方文件。建议如果Context无提供相应的方法或属性时，才使用此对象。

## Response

http.ResponseWriter，用于设置HTTP响应相关状态码、响应头与响应数据，context有各类函数操作此对象，一般无需通过直接操作此对象。

## Committed

是否已将响应数据返回（状态码、数据等已写入至Response），除非需要单独处理数据的响应，否则不要设置此属性。

## ID

Context ID，如果有设置Elton.GenerateID，则在每次接收到请求，创建Context之后，调用`GenerateID`生成，一般用于日志或者统计中唯一标识当前请求。

## Route

当前对应的路由。

## Next

next函数，此函数会在获取请求时自动生成，无需调整。如果是测试是直接NewContext，则需要设置对应的Next方法。

## Params

路由参数对象，提供获取路由中参数方法，不需要直接使用此对象，使用Context中的Param方法获取则可

## StatusCode

HTTP响应码，设置HTTP请求处理结果的响应码。

## Body

HTTP响应数据，此属性为interface{}，因此可以设置不同的数据类型（与koa类似）。注意：设置Body之后，还需要使用中间件`responder`来将此属性转换为字节，并设置相应的`Content-Type`，此中间件主要将各类的struct转换为json，对于具体的实现可以查阅代码，或者自己实现相应的responder。

## BodyBuffer

HTTP的响应数据缓冲（字节），此数据为真正返回的响应体，不建议直接赋值此属性，而应该则responder中间件将Body转换为字节(BodyBuffer)，并写入相应的`Content-Type`。

## RequestBody

HTTP请求体，对于`POST`，`PUT`以及`PATCH`提交数据的请求，此字段用于保存请求体。注意：默认Elton中并未从请求中读取相应的请求体，需要使用`body_parser`中间件来获取或者自定义相应的中间件。

## RemoteAddr

获取请求客户端的IP，直接获取连接的客户端，不会从请求头中获取。

**Example**
```go
package main

import (
	"log"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		log.Println(c.RemoteAddr())
		c.Body = "Hello, World!"
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## RealIP

获取客户端的真实IP，先判断请求头是否有`X-Forwarded-For`，如果没有再取`X-Real-Ip`，都没有则从连接IP中取。

**Example**
```go
package main

import (
	"log"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		log.Println(c.RealIP())
		c.Body = "Hello, World!"
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## ClientIP

获取客户端真实IP，其获取方式与`RealIP`类似，但在获取到IP时，先判断是否公网IP，如果非公网IP，则继续获取下一符合条件的IP。

**Example**
```go
package main

import (
	"log"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		log.Println(c.ClientIP())
		c.Body = "Hello, World!"
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## Param

获取路由的参数。

**Example**
```go
// curl http://127.0.0.1:3000/users/me
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/users/{type}", func(c *elton.Context) (err error) {
		c.Body = c.Param("type")
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## QueryParam

获取query的参数值，此函数返回的并非字符串数组，只取数组的第一个，如果query中的相同的key的使用，请直接使用`Request.URL.Query()`来获取。

**Example**
```go
// curl http://127.0.0.1:3000/?type=vip&count=10
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.Body = c.QueryParam("type")
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## Query

获取请求的querystring，此函数返回的query对象为map[string]string，不同于原有的map[string][]string，因为使用相同的key的场景不多，因此增加此函数方便使用。如果有相同的key的场景，请直接使用`Request.URL.Query()`来获取。

**Example**
```go
// curl http://127.0.0.1:3000/?type=vip&count=10
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.Body = c.Query()
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## Redirect

重定向当前请求。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.Body = "Hello, World!"
		return
	})
	e.GET("/redirect", func(c *elton.Context) (err error) {
		err = c.Redirect(301, "/")
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## Set/Get

设置保存的值至context，在context的生命周期内有效，调用Get方法则可获取保存的值。还有各类基本类型数据的快捷获取方法，将保存的数据转换为对应的类型并返回，若该数据不存在或类型不匹配，则返回默认值。支持的方法如下：`GetInt`, `GetInt64`, `GetString`, `GetBool`, `GetFloat32`, `GetFloat64`, `GetTime`, `GetDuration`, `GetStringSlice`。

**Example**
```go
package main

import (
	"math/rand"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.Use(func(c *elton.Context) error {
		c.Set("id", rand.Int())
		return c.Next()
	})

	e.GET("/", func(c *elton.Context) (err error) {
		value, _ := c.Get("id")
		c.Body = value
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## GetRequestHeader

从HTTP请求头中获取相应的值。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.Body = c.GetRequestHeader("User-Agent")
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## SetRequestHeader

设置HTTP请求头的值，如果该值已存在，则覆盖。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.SetRequestHeader("User-Agent", "go-agent")
		c.Body = c.GetRequestHeader("User-Agent")
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## AddRequestHeader

添加HTTP请求头的值，它不会覆盖原有值，而是添加。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.AddRequestHeader("User-Agent", "go-agent")
		c.Body = c.Request.Header
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## Context

Get context of request.

## WithContext

Set request with context.


## Header

返回HTTP响应头。

**Example**
```go
package main

import (
	"math/rand"
	"strconv"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.SetHeader("X-Response-Id", strconv.Itoa(rand.Int()))
		c.Body = c.Header()
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## GetHeader

从HTTP响应头中获取相应的值。

**Example**
```go
package main

import (
	"math/rand"
	"strconv"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.SetHeader("X-Response-Id", strconv.Itoa(rand.Int()))
		c.Body = c.GetHeader("X-Response-Id")
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## SetHeader

设置HTTP响应头的值，如果该值已存在，则覆盖。

**Example**
```go
package main

import (
	"math/rand"
	"strconv"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.SetHeader("X-Response-Id", strconv.Itoa(rand.Int()))
		c.Body = c.GetHeader("X-Response-Id")
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## AddHeader

添加HTTP响应头的值，它不会覆盖原有值，而是添加。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.AddHeader("X-Response-Id", "1")
		c.AddHeader("X-Response-Id", "2")
		c.Body = c.Header()
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## MergeHeader

合并HTTP头

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		h := make(http.Header)
		h.Add("X-Response-Id", "1")
		h.Add("X-Response-Id", "2")
		c.MergeHeader(h)
		c.Body = c.Header()
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## ResetHeader

重置HTTP响应头的所有值。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.AddHeader("X-Response-Id", "1")
		c.AddHeader("X-Response-Id", "2")
		c.ResetHeader()
		c.Body = c.Header()
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## Cookie/AddCookie

Cookie方法从HTTP请求头中获取cookie，AddCookie则添加cookie至HTTP响应头。

**Example**
```go
package main

import (
	"math/rand"
	"net/http"
	"strconv"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		cookie, _ := c.Cookie("jt")
		if cookie == nil {
			_ = c.AddCookie(&http.Cookie{
				Name:  "jt",
				Value: strconv.Itoa(rand.Int()),
			})
		}
		c.Body = cookie
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## SignedCookie/AddSignedCookie

SignedCookie则会根据初始化Elton时配置的Keys来校验cookie与sig cookie是否符合，符合才返回。AddSignedCookie则根据当前的Cookie以及初化Elton时配置的Keys再生成一个校验cookie(Name为当前Cookie的Name + ".sig")。

**Example**
```go
package main

import (
	"math/rand"
	"net/http"
	"strconv"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()
	e.SignedKeys = new(elton.RWMutexSignedKeys)
	e.SignedKeys.SetKeys([]string{
		"secret key",
	})

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		cookie, _ := c.SignedCookie("jt")
		if cookie == nil {
			_ = c.AddSignedCookie(&http.Cookie{
				Name:  "jt",
				Value: strconv.Itoa(rand.Int()),
			})
		}
		c.Body = cookie
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## SendFile

读取文件并响应，在获取时根据文件的修改时间生成`Last-Modified`，并设置`Content-Length`与`Content-Type`，数据以Pipe的形式响应。

**Example**
```go
package main

import (
	"math/rand"
	"net/http"
	"strconv"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		return c.SendFile("index.html")
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## NoContent

设置HTTP请求的响应状态码为204，响应体为空。

**Example**
```go
// curl 'http://127.0.0.1:3000/' -v
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.NoContent()
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## NotModified

设置HTTP请求的响应状态码为304，响应体为空。注意此方法判断是否客户端的缓存数据与服务端的响应数据一致再使用，建议使用中间件`fresh`处理则可。

**Example**
```go
// curl 'http://127.0.0.1:3000/' -v
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.NotModified()
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```


## NoCache

设置HTTP响应头的`Cache-Control: no-cache`，建议使用全局中间件设置所有请求默认为no-cache，对于需要调整的路由则在处理函数中单独设置。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())
	e.Use(func(c *elton.Context) error {
		c.NoCache()
		return c.Next()
	})

	e.GET("/", func(c *elton.Context) (err error) {
		c.Body = "Hello, World!"
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## NoStore

设置HTTP响应头的`Cache-Control: no-store`，用于不希望客户端保存的请求，如验证码等一次性请求。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.NoStore()
		c.Body = "Hello, World!"
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## CacheMaxAge

设置HTTP响应头的`Cache-Control: public, max-age=x, s-maxage=y`，支持动态参加设置，第一个参数为`max-age`，第二个参数为`s-maxage`。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.CacheMaxAge(time.Minute, 10 * time.Second)
		c.Body = "Hello, World!"
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```


## Created

设置HTTP请求的响应码为201，并设置响应体。

**Example**
```go
// curl -XPOST 'http://127.0.0.1:3000/' -v
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.POST("/", func(c *elton.Context) (err error) {
		c.Created(map[string]string{
			"account": "tree.xie",
		})
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## SetContentTypeByExt

通过文件（文件后缀）设置Content-Type。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.SetContentTypeByExt(".html")
		c.Body = `<html>
			<body>
				<p>Hello, World!</p>
			</body>
		</html>`
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## DisableReuse

禁止context复用，如果context在所有handler执行之后，还需要使用（如设置了超时出错，但无法对正在执行的handler中断，此时context还在使用中），则需要调用此函数禁用context的复用，除非有必要不建议禁止复用。

**Example**
```go
package main

import (
	"fmt"
	"time"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		go func() {
			time.Sleep(time.Second)
			fmt.Println(c)
		}()
		c.DisableReuse()
		c.Body = "Hello, World!"
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## Pass

将当前context的处理pass给另一个Elton实例，设置Committed为true，此实例的所有处理函数均不再使用处理此context。

## Pipe

将当前Reader pipe向Response，用于流式输出响应数据，节省内存使用。

**Example**
```go
package main

import (
	"bytes"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		buf := new(bytes.Buffer)
		for i := 0; i < 1000; i++ {
			buf.WriteString("Hello, World!\n")
		}
		_, _ = c.Pipe(buf)
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```