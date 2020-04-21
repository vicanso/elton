---
description: 各类常用的中间件
---

# Middlewares

- [basic auth](#basic-auth) HTTP Basic Auth，建议只用于内部管理系统使用
- [body parser](#body-parser) 请求数据的解析中间件，支持`application/json`以及`application/x-www-form-urlencoded`两种数据类型
- [compress](#compress) 数据压缩中间件，默认仅支持gzip。如果需要支持更多的压缩方式，如brotli、snappy、s2、zstd以及lz4，可以使用[elton-compress](https://github.com/vicanso/elton-compress)，也可根据需要增加相应的压缩处理
- [concurrent limiter](#concurrent-limiter) 根据指定参数限制并发请求，可用于订单提交等防止重复提交或限制提交频率的场景
- [error handler](#error-handler) 用于将处理函数的Error转换为对应的响应数据，如HTTP响应中的状态码(40x, 50x)，对应的出错类别等，建议在实际使用中根据项目自定义的Error对象生成相应的响应数据
- [etag](#etag) 用于生成HTTP响应数据的ETag
- [fresh](#fresh) 判断HTTP请求是否未修改(Not Modified)
- [json picker](https://github.com/vicanso/elton-json-picker) 用于从响应的JSON中筛选指定字段
- [jwt](https://github.com/vicanso/elton-jwt) jwt中间件
- [logger](#logger) 生成HTTP请求日志，支持从请求头、响应头中获取相应信息
- [proxy](#proxy) Proxy中间件，可定义请求转发至其它的服务
- [recover](#recover) 捕获程序的panic异常，避免程序崩溃
- [responder](#responder) 响应处理中间件，用于将`Context.Body`(interface{})转换为对应的JSON数据并输出。如果系统使用xml等输出响应数据，可参考此中间件实现interface{}至xml的转换
- [router-concurrent-limiter](#router-concurrent-limiter) 路由并发限制中间件，可以针对路由限制并发请求量。
- [session](https://github.com/vicanso/elton-session) Session中间件，默认支持保存至redis或内存中，也可自定义相应的存储
- [stats](#stats) 请求处理的统计中间件，包括处理时长、状态码、响应数据长度、连接数等信息
- [static serve](#static-serve) 静态文件处理中间件，默认支持从目录中读取静态文件或实现StaticFile的相关接口，从[packr](github.com/gobuffalo/packr/v2)或者数据库(mongodb)等读取文件
- [tracker](#tracker) 可以用于在POST、PUT等提交类的接口中增加跟踪日志，此中间件将输出QueryString，Params以及RequestBody部分，并能将指定的字段做"***"的处理，避免输出敏感信息

## basic auth

HTTP basic auth中间件，提供简单的认证方式，建议只用于内部管理系统。

**Example**
```go
package main

import (
	"bytes"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
	"github.com/vicanso/hes"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewBasicAuth(middleware.BasicAuthConfig{
		Validate: func(account, pwd string, c *elton.Context) (bool, error) {
			if account == "tree.xie" && pwd == "password" {
				return true, nil
			}
			if account == "n" {
				return false, hes.New("account is invalid")
			}
			return false, nil
		},
	}))

	e.GET("/", func(c *elton.Context) (err error) {
		c.BodyBuffer = bytes.NewBufferString("hello world")
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## body parser

解析HTTP请求接收到的数据，默认支持`json`与`form`的提交，可以根据应用场景增加各类Decoder以支持更多的数据类型。


### NewDefaultBodyParser

创建一个默认的body parser中间件，它包括gzip与json的处理。
```go
e.Use(middleware.NewDefaultBodyParser())
```

### NewGzipDecoder

创建一个gzip数据的decoder

```go
conf := middleware.BodyParserConfig{}
conf.AddDecoder(middleware.NewGzipDecoder())
e.Use(middleware.NewBodyParser(conf))
```

### NewJSONDecoder

创建一个json数据的decoder

```go
conf := middleware.BodyParserConfig{}
conf.AddDecoder(middleware.NewJSONDecoder())
e.Use(middleware.NewBodyParser(conf))
```

### NewFormURLEncodedDecoder

创建一个form数据的decoder

```go
conf := middleware.BodyParserConfig{
	ContentTypeValidate: middleware.DefaultJSONAndFormContentTypeValidate
}
conf.AddDecoder(middleware.NewFormURLEncodedDecoder())
e.Use(middleware.NewBodyParser(conf))
```

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

	e.Use(middleware.NewDefaultBodyParser())

	e.POST("/user/login", func(c *elton.Context) (err error) {
		c.BodyBuffer = bytes.NewBuffer(c.RequestBody)
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## compress

响应数据压缩中间件，可对特定数据类型、数据长度的响应数据做压缩处理。默认支持`gzip`压缩，可扩展更多的压缩方式，如：brotli，zstd等。

### Compressor

实现自定义的压缩主要实现三下方法：

- `Accept` 判断该压缩是否支持该压缩，根据请求头以及响应数据大小
- `Compress` 数据压缩方法
- `Pipe` 数据Pipe处理

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

	e.Use(middleware.NewDefaultCompress())

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) error {
		b := new(bytes.Buffer)
		for i := 0; i < 1000; i++ {
			b.WriteString("Hello, World!")
		}
		c.Body = &struct {
			Message string
		}{
			b.String(),
		}
		return nil
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```


## concurrent limiter

并发请求限制，可以通过指定请求的参数，如IP、query的字段或者body等获取，限制同时并发性的提交请求，主要用于避免相同的请求多次提交。指定的Key分为以下几种：

- `:ip` 客户的RealIP
- `h:key` 从HTTP请求头中获取key的值
- `q:key` 从HTTP的query中获取key的值
- `p:key` 从路由的params中获取key的值
- 其它的则从HTTP的Post data中获取key的值（只支持json)

**Example**
```go
package main

import (
	"bytes"
	"sync"
	"time"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {

	e := elton.New()
	m := new(sync.Map)
	limit := middleware.NewConcurrentLimiter(middleware.ConcurrentLimiterConfig{
		Keys: []string{
			":ip",
			"h:X-Token",
			"q:type",
			"p:id",
			"account",
		},
		Lock: func(key string, c *elton.Context) (success bool, unlock func(), err error) {
			_, loaded := m.LoadOrStore(key, true)
			// the key not exists
			if !loaded {
				success = true
				unlock = func() {
					m.Delete(key)
				}
			}
			return
		},
	})

	e.POST("/login", limit, func(c *elton.Context) (err error) {
		time.Sleep(3 * time.Second)
		c.BodyBuffer = bytes.NewBufferString("hello world")
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## error handler

出错转换处理，用于将出错转换为json或text出错响应，建议在controller中对处理出错的自定义出错类型，使用出错中间件将相应的出错信息转换输出。

**Example**
```go
package main

import (
	"errors"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {

	e := elton.New()
	e.Use(middleware.NewDefaultError())

	e.GET("/", func(c *elton.Context) (err error) {
		err = errors.New("abcd")
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## etag

根据响应数据生成HTTP响应头的ETag。

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
	e.Use(middleware.NewDefaultETag())

	e.GET("/", func(c *elton.Context) (err error) {
		c.BodyBuffer = bytes.NewBufferString("abcd")
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## fresh

根据HTTP请求头与响应头判断是否未修改(304 Not Modified)。

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
	e.Use(middleware.NewDefaultFresh())
	e.Use(middleware.NewDefaultETag())

	e.GET("/", func(c *elton.Context) (err error) {
		c.BodyBuffer = bytes.NewBufferString("abcd")
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## logger

Logger中间件，支持从请求头、响应头等获取信息，支持以下标签：

{host} {remote} {real-ip} {method} {path} {proto} {query} {scheme} {uri} {referer} {userAgent} {size} {size-human} {status} {payload-size} {payload-size-human}

其中`size-human`以及`payload-size-human`是将字节格式化带单位的形式，主要讲解三种字段的使用：

- `~cookie` 表示获取cookie的值，必须以~开头，后面的表示cookie的key
- `>header` 表示获取请求头的值，必须以>开头，后面表示header的key
- `<header` 表示获取响应头的值，必须以<开头，后面表示header的key

预定义了四种格式化模板(建议使用时自定义日志模板)：
- `LoggerCombined`: `{remote} {when-iso} "{method} {uri} {proto}" {status} {size-human} "{referer}" "{userAgent}"`
- `LoggerCommon`: `{remote} {when-iso} "{method} {uri} {proto}" {status} {size-human}`
- `LoggerShort`: `{remote} {method} {uri} {proto} {status} {size-human} - {latency-ms} ms`
- `LoggerTiny`: `{method} {url} {status} {size-human} - {latency-ms} ms`

**Example**
```go
package main

import (
	"fmt"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	// panic处理
	e.Use(middleware.NewRecover())
	e.Use(middleware.NewLogger(middleware.LoggerConfig{
		Format: middleware.LoggerCombined,
		OnLog: func(str string, _ *elton.Context) {
			fmt.Println(str)
		},
	}))

	// 响应数据转换为json
	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) error {
		c.Body = &struct {
			Message string `json:"message,omitempty"`
		}{
			"Hello, World!",
		}
		return nil
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## proxy

Proxy中间件，可以将指定的请求转发至另外的服务，并可重写url。

**Example**
```go
package main

import (
	"net/url"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	target, _ := url.Parse("https://www.baidu.com")

	e.GET("/*", middleware.NewProxy(middleware.ProxyConfig{
		// proxy done will call this function
		Done: func(c *elton.Context) {

		},
		// http request url rewrite
		Rewrites: []string{
			"/api/*:/$1",
		},
		Target: target,
		// change the request host
		Host: "www.baidu.com",
	}))

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## recover

Recover中间件，用于捕获各种panic异常，避免程序异常退出。可根据应用场景编写自定义的panic处理，如增加告警等。

**Example**
```go
package main

import (
	"errors"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewRecover())

	e.GET("/", func(c *elton.Context) (err error) {
		panic(errors.New("abcd"))
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## responder

用于将Body转换为对应的字节数据，并设置响应头。默认的处理为将struct(map)转换为json，对于不同的应用可以指定Marshal与ContentType来实现自定义响应。

- `ResponderConfig.Marshal` 自定义的Marshal函数，默认为`json.Marshal`
- `ResponderConfig.ContentType` 自定义的ContentType，默认为`application/json; charset=UTF-8`

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

	// {"name":"tree.xie","id":123}
	e.GET("/", func(c *elton.Context) (err error) {
		c.Body = &struct {
			Name string `json:"name"`
			ID   int    `json:"id"`
		}{
			"tree.xie",
			123,
		}
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## router concurrent limiter

路由限制中间件，可以指定路由的并发访问数量，建议使用`NewLocalLimiter`每个实例的限制分开，主要是用于避免某个接口并发过高导致系统不稳定。

**Example**
```go
package main

import (
	"bytes"
	"time"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewRCL(middleware.RCLConfig{
		Limiter: middleware.NewLocalLimiter(map[string]uint32{
			"GET /users/me": 2,
		}),
	}))

	e.GET("/users/me", func(c *elton.Context) (err error) {
		time.Sleep(time.Second)
		c.BodyBuffer = bytes.NewBufferString(`{
			"account": "tree",
			"name": "tree.xie"
		}`)
		return nil
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## stats

HTTP请求的统计中间件，可以根据此中间件将http请求的各类统计信息写入至统计数据库，如：influxdb等，方便根据统计来优化性能以及监控。

**Example**
```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewStats(middleware.StatsConfig{
		OnStats: func(info *middleware.StatsInfo, _ *elton.Context) {
			buf, _ := json.Marshal(info)
			fmt.Println(string(buf))
		},
	}))

	e.GET("/", func(c *elton.Context) (err error) {
		c.BodyBuffer = bytes.NewBufferString("abcd")
		return
	})
	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## static serve

静态文件处理中间件，默认支持通过目录访问，在实例使用中可以根据需求实现接口以使用各类不同的存储方式，如`packr`打包或mongodb存储等。

**Example**
```go
package main

import (
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	sf := new(middleware.FS)
	// static file route
	e.GET("/*", middleware.NewStaticServe(sf, middleware.StaticServeConfig{
		Path: "/tmp",
		// 客户端缓存一年
		MaxAge: 365 * 24 * 3600,
		// 缓存服务器缓存一个小时
		SMaxAge:             60 * 60,
		DenyQueryString:     true,
		DisableLastModified: true,
		// packr不支持Stat，因此需要用强ETag
		EnableStrongETag: true,
	}))

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

使用packr打包前端应用程序，通过static serve提供网站静态文件访问：

**Example**
```go
package main

import (
	"bytes"
	"io"
	"os"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

var (
	box = packr.New("asset", "./")
)

type (
	staticFile struct {
		box *packr.Box
	}
)

func (sf *staticFile) Exists(file string) bool {
	return sf.box.Has(file)
}
func (sf *staticFile) Get(file string) ([]byte, error) {
	return sf.box.Find(file)
}
func (sf *staticFile) Stat(file string) os.FileInfo {
	return nil
}
func (sf *staticFile) NewReader(file string) (io.Reader, error) {
	buf, err := sf.Get(file)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(buf), nil
}

func main() {
	e := elton.New()

	sf := &staticFile{
		box: box,
	}

	// static file route
	e.GET("/static/*", middleware.NewStaticServe(sf, middleware.StaticServeConfig{
		// 客户端缓存一年
		MaxAge: 365 * 24 * 3600,
		// 缓存服务器缓存一个小时
		SMaxAge:             60 * 60,
		DenyQueryString:     true,
		DisableLastModified: true,
	}))

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

## tracker

用于在客户提交类的请求添加跟踪日志，可输出query、body以及params等信息，并可设置正则匹配将关键数据加*处理。

**Example**
```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
)

func main() {
	e := elton.New()

	loginTracker := middleware.NewTracker(middleware.TrackerConfig{
		OnTrack: func(info *middleware.TrackerInfo, _ *elton.Context) {
			buf, _ := json.Marshal(info)
			fmt.Println(string(buf))
		},
	})

	e.Use(func(c *elton.Context) error {
		c.RequestBody = []byte(`{
			"account": "tree.xie",
			"password": "123456"
		}`)
		return c.Next()
	})

	e.POST("/user/login", loginTracker, func(c *elton.Context) (err error) {
		c.SetHeader(elton.HeaderContentType, elton.MIMEApplicationJSON)
		c.BodyBuffer = bytes.NewBuffer(c.RequestBody)
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```