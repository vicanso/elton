---
description: elton实例的相关方法说明
---

# Application

## New

创建一个elton实例，并初始化相应的http.Server。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## NewWithoutServer

创建一个Elton的实例，并未初始化相应的http.Server，主要用于需要自定义http server的场景，如各类超时设置等。

**Example**
```go
package main

import (
	"net/http"
	"time"

	"github.com/vicanso/elton"
)

func main() {
	e := elton.NewWithoutServer()
	s := &http.Server{
		Handler:      e,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	e.Server = s
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## ErrorHandler

自定义的Error处理，若路由处理过程中返回Error，则会触发此调用，若未指定此处理函数，则使用默认的处理，简单的输出`err.Error()`。

注意若在处理过程中返回的Error已被处理（如Error Handler），则并不会触发此出错调用，尽量使用中间件将Error转换为相应的输出，如JSON。

**Example**
```go
package main

import (
	"log"
	"net/http"

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

func main() {
	e := elton.New()

	e.ErrorHandler = func(c *elton.Context, err error) {
		if err != nil {
			log.Printf("未处理异常，url:%s, err:%v", c.Request.RequestURI, err)
		}
		he, ok := err.(*hes.Error)
		if ok {
			c.Response.WriteHeader(he.StatusCode)
			c.Response.Write([]byte(he.Message))
		} else {
			c.Response.WriteHeader(http.StatusInternalServerError)
			c.Response.Write([]byte(err.Error()))
		}
	}

	e.GET("/", func(c *elton.Context) (err error) {
		return hes.New("abcd")
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## NotFoundHandler

未匹配到相应路由时的处理，当无法获取到相应路由时，则会调用此函数（未匹配相应路由时，所有的中间件也不会被调用）。如果有相关统计需要或者自定义的404页面，则可调整此函数，否则可不设置使用默认处理(返回404 Not Found)。

**Example**
```go
package main

import (
	"log"
	"net/http"

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

func main() {
	e := elton.New()

	e.NotFoundHandler = func(resp http.ResponseWriter, req *http.Request) {
		// 可增加统计，方便分析404的处理是被攻击还是接口调用错误
		log.Printf("404，url:%s", req.RequestURI)
		resp.WriteHeader(http.StatusNotFound)
		resp.Write([]byte("Custom not found"))
	}

	e.GET("/ping", func(c *elton.Context) (err error) {
		return hes.New("abcd")
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## GenerateID

ID生成函数，用于每次请求调用时，生成唯一的ID值。

**Example**
```go
package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/oklog/ulid"
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.GenerateID = func() string {
		t := time.Now()
		entropy := rand.New(rand.NewSource(t.UnixNano()))
		return ulid.MustNew(ulid.Timestamp(t), entropy).String()
	}

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		log.Println(c.ID)
		c.Body = c.ID
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## EnableTrace

是否启用调用跟踪，设置此参数为true，则会记录每个Handler的调用时长，建议使用时对全局中间件设定名称。

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

	e.EnableTrace = true
	e.OnTrace(func(c *elton.Context, traceInfos elton.TraceInfos) {
		log.Println(traceInfos[0])
		// 设置HTTP响应头：Server-Timing
		c.ServerTiming(traceInfos, "elton-")
	})

	fn := middleware.NewDefaultResponder()
	// 自定义该中间件的名称，如果设置为"-"，则忽略该中间件
	e.SetFunctionName(fn, "responder")
	e.Use(fn)

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

## SignedKeys

用于生成带签名的cookie的密钥，基于[keygrip](https://github.com/vicanso/keygrip)来生成与校验是否合法。使用`SignedCookie`获取cookie时，会校验cookie的合法性，而`AddSignedCookie`则会在添加cookie的同时再另外添加相对应的sig cookie。

**Example**
```go
package main

import (
	"bytes"
	"net/http"

	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()

	e.SignedKeys = new(elton.RWMutexSignedKeys)
	// 密钥，用于生成cookie的签名
	e.SignedKeys.SetKeys([]string{
		"secret key",
	})

	e.GET("/", func(c *elton.Context) (err error) {
		cookie, _ := c.SignedCookie("jt")
		// 如果该cookie不存在
		if cookie == nil {
			// 设置signed cookie
			err = c.AddSignedCookie(&http.Cookie{
				Name:  "jt",
				Value: "abcd",
			})
			if err != nil {
				return
			}
		}

		c.BodyBuffer = bytes.NewBufferString("Hello, World!")
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## ListenAndServe

设定监听地址，并调用http.Server的`ListenAndServe`提供HTTP服务。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## ListenAndServeTLS

设定监听地址，并调用http.Server的`ListenAndServeTLS`提供HTTPS服务。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
)

func main() {
    // 加密证书相关路径
    certFile := "~/cert/cert"
    keyFile := "~/cert/key"
	e := elton.New()
	err := e.ListenAndServeTLS(":3000", certFile, keyFile)
	if err != nil {
		panic(err)
	}
}
```

## Close

关闭服务，调用http.Server的`Close`方法。

**Example**
```go
package main

import (
	"time"

	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()

	go func() {
		time.Sleep(5 * time.Second)
		e.Close()
	}()

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## GracefulClose

优雅的关闭当前服务，首先将实例的状态设置为`StatusClosing`，此时所有新的请求都将直接出错，在等于指定时间后，则调用http.Server的`Close`方法。

**Example**
```go
package main

import (
	"time"

	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()

	go func() {
		e.GracefulClose(10 * time.Second)
	}()

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## Handle

添加Handler的处理函数，配置请求的Method与Path，添加相应的处理函数。Elton还支持GET，POST，PUT，PATCH，DELETE，HEAD，TRACE以及OPTIONS的方法，这几个方法与`Handle`一致，Method则为相对应的处理，以及可使用ALL来指定支持所有的http method。

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

	noop := func(c *elton.Context) error {
		return c.Next()
	}

	e.Handle("GET", "/", noop, func(c *elton.Context) (err error) {
		c.Body = "Hello, World!"
		return
	})

	e.POST("/users/{type}", func(c *elton.Context) (err error) {
		c.Body = "OK"
		return
	})

	e.GET("/files/*", func(c *elton.Context) (err error) {
		c.Body = "file content"
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## Use

添加全局中间件处理函数，对于所有路由都需要使用到的中间件，则使用此函数添加，若非所有路由都使用到，可以只添加到相应的Group或者就单独添加至Handler。特别需要注意的是，如session之类需要读取数据库的，如非必要，不要使用全局中间件形式。

**Example**
```go
package main

import (
	"log"
	"time"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	// 记录HTTP请求的时间、响应码
	e.Use(func(c *elton.Context) (err error) {
		startedAt := time.Now()
		req := c.Request
		err = c.Next()
		log.Printf("%s %s %d use %s", req.Method, req.URL.RequestURI(), c.StatusCode, time.Since(startedAt).String())
		return err
	})

	e.Use(middleware.NewDefaultResponder())

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

## Pre

添加全局前置中间件处理函数，对于所有请求都会调用（包括无匹配路由的请求）。

**Example**
```go
package main

import (
	"net/http"
	"strings"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()
	// 如果url以/api开头，则替换
	urlPrefix := "/api"
	e.Pre(func(req *http.Request) {
		path := req.URL.Path
		if strings.HasPrefix(path, urlPrefix) {
			req.URL.Path = path[len(urlPrefix):]
		}
	})
	e.Use(middleware.NewDefaultResponder())

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

## AddGroup

将group中的所有路由处理添加至Elton。

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
	userGroup := elton.NewGroup("/users", func(c *elton.Context) error {
		return c.Next()
	})
	userGroup.GET("/me", func(c *elton.Context) error {
		c.Body = "nickname"
		return nil
	})

	e.AddGroup(userGroup)
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## OnError

添加Error的监听函数，如果当任一Handler的处理返回Error，并且其它的Handler并未将此Error处理(建议使用专门的中间件处理出错)，则会触发error事件，建议使用此事件来监控程序未处理异常。

**Example**
```go
package main

import (
	"errors"
	"log"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.OnError(func(c *elton.Context, err error) {
		// 发送邮件告警等
		log.Println("error: " + err.Error())
	})

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) (err error) {
		c.Body = "Hello, World!"
		return
	})
	// 由于未设置公共的出错处理中间件，此error会触发事件
	// 实际使用中，需要添加公共的error handler来处理，error事件只用于异常的出错
	e.GET("/error", func(c *elton.Context) (err error) {
		return errors.New("abcd")
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```