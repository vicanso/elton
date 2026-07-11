---
description: 各类常用的中间件
---

# Middlewares

- [recommended](#recommended) 常用全局中间件栈（一键 `e.Use(middleware.Recommended()...)`）
- [basic auth](#basic-auth) HTTP Basic Auth，建议只用于内部管理系统使用
- [body parser](#body-parser) 请求数据解析，支持 `application/json` 与 `application/x-www-form-urlencoded`
- [cache](#cache) HTTP 缓存，基于响应头 `Cache-Control`，可配合 br/gzip/zstd 压缩后写入 store
- [compress](#compress) 响应压缩；内置 gzip / brotli / zstd，其它算法（如 snappy、lz4）可实现 `Compressor` 接口扩展（见 [自定义压缩](./custom_compress.md)）
- [cors](#cors) 跨域（含预检短路）
- [global concurrent limiter](#global-concurrent-limiter) 全局在途请求数限制
- [concurrent limiter](#concurrent-limiter) 按 IP/头/query/body 等维度限制并发，防重复提交
- [error handler](#error-handler) 将处理函数返回的 `error` 转为 HTTP 状态码与响应体（内置支持 [hes.Error](https://github.com/vicanso/hes)）
- [etag](#etag) 生成响应 ETag
- [fresh](#fresh) 判断是否可返回 304 Not Modified
- [json picker](https://github.com/vicanso/elton-json-picker)（外部）从响应 JSON 中筛选字段
- [jwt](https://github.com/vicanso/elton-jwt)（外部）JWT 中间件
- [logger](#logger) 请求日志，可从请求/响应头取值
- [proxy](#proxy) 反向代理
- [recover](#recover) 捕获 panic，避免进程崩溃
- [renderer](#renderer) 模板渲染为 HTML
- [request id](#request-id) 请求 ID（透传或生成，写入响应头与 context）
- [responder](#responder) 将 `Context.Body`（`any`）转为 JSON 等并写入 `BodyBuffer`；XML 等可自定义 marshal
- [response-size-limiter](#response-size-limiter) 限制响应体最大长度
- [router-concurrent-limiter](#router-concurrent-limiter) 按路由限制并发
- [session](https://github.com/vicanso/elton-session)（外部）Session，默认可存内存，可自定义存 redis 等
- [stats](#stats) 请求统计（耗时、状态码、响应长度等）
- [static serve](#static-serve) 静态文件；支持 OS 目录、`embed.FS`、自实现 `StaticFile` / encoding FS
- [timeout](#timeout) 请求处理截止时间（依赖 `c.Context()` 协作取消）
- [tracker](#tracker) 提交类接口跟踪日志（Query/Params/Body，支持字段脱敏）

## recommended

JSON API 常用全局栈，等价于 README Hello World 中的中间件组合，并加上 RequestID：

`Recover → Error → RequestID → BodyParser → Fresh → ETag → Responder`

```go
e := elton.New()
e.Use(middleware.Recommended()...)
// 按需再加：CORS、Timeout、Compress、Logger
e.Use(middleware.NewDefaultTimeout(5 * time.Second))
e.Use(middleware.NewDefaultCORS())
```

完整示例见仓库 [`examples/`](../examples/)。

## basic auth

HTTP basic auth中间件，提供简单的认证方式，建议只用于内部管理系统。

**Example**
```go
package main

import (
	"bytes"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
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

解析HTTP请求接收到的数据，支持`json`与`form`的提交，可以根据应用场景增加各类Decoder以支持更多的数据类型，如提交数据的`gzip`解压等。


### NewDefaultBodyParser

创建一个默认的body parser中间件，它包括`gzip`与`json`的处理。首先根据提交数据的`Content-Encoding`是否为`gzip`，如果是则先解压，再判断数据是否`json`。

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

创建一个form数据的decoder(不建议使用)

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

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
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

## cache

缓存中间件，对于`GET`与`HEAD`的请求，根据其`Cache-Control`判断是否可缓存，若可缓存则将数据缓存至store中，下次相同的请求直接从缓存中读取。缓存数据可指定数据压缩后缓存，并响应时根据客户端自动返回压缩或未压缩数据。需要注意当前基本所有浏览器均支持br压缩，但是浏览器只在https模式下才会设置支持br，因此服务仅运行在http上，则建议使用gzip压缩。

- 请求的缓存key默认为`Method` + `RequestURI`
- `fetch`状态则表示无缓存时的请求，获取响应数据后判断是否可缓存，如果可缓存则设置缓存数据(状态:hit,数据:响应头及响应数据)，不可缓存则设置缓存数据(状态:hit-for-pass,数据:空)
- `hit-for-pass`状态表示该请求有相应缓存，但该缓存表示该请求不可读取缓存
- `hit`状态表示该请求有相应缓存，则缓存数据可用，直接使用缓存返回客户端
- 如果有设置压缩，缓存数据若符合压缩条件则压缩后缓存，若不符合，则缓存原始数据。响应时需要客户端是否可接受压缩数据，若可以则直接返回压缩数据，若不可以则解压后返回


**Example**
```go
package main

import (
	"bytes"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {
	e := elton.New()
	// 使用redis实现的store
	e.Use(middleware.NewDefaultCache(redisStore))

	e.GET("/", func(c *elton.Context) (err error) {
		c.CacheMaxAge(time.Minute)
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

响应压缩中间件，可按 `Content-Type`、体长度与客户端 `Accept-Encoding` 选择算法。

- **内置**：`NewGzipCompressor()`、`NewBrCompressor()`、`NewZstdCompressor()`
- **默认**：`NewDefaultCompress()` 仅启用 gzip；多算法请用 `NewCompressConfig(...)` / `NewCompress`
- **扩展**：实现 `Compressor` 接口即可接入 snappy、lz4 等，见 [自定义压缩](./custom_compress.md)

### Compressor

自定义压缩需实现：

- `Accept`：是否对当前请求/体长启用该编码
- `Compress`：缓冲数据压缩（可选 level 参数）
- `Pipe`：流式 `Body`（`io.Reader`）压缩写出

**Example**
```go
package main

import (
	"bytes"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {

	e := elton.New()

	// 同时启用 gzip / br / zstd（按 Accept-Encoding 协商）
	e.Use(middleware.NewCompress(middleware.NewCompressConfig(
		middleware.NewGzipCompressor(),
		middleware.NewBrCompressor(),
		middleware.NewZstdCompressor(),
	)))

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


## global concurrent limiter

全局在途请求数限制，用于保护进程不被瞬时流量打满。

**Max 语义（重要）**：在途计数 `Add(1)` 之后若 `value >= Max` 则拒绝。因此配置 `Max: N` 时，**实际允许的最大并发为 `N - 1`**（与 v1 一致）。例如希望最多 1000 路同时处理，应设 `Max: 1001`，或按「阈值」理解并在容量规划中预留 1。

**Example**
```go
package main

import (
	"bytes"
	"time"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {

	e := elton.New()
	e.Use(middleware.NewGlobalConcurrentLimiter(middleware.GlobalConcurrentLimiterConfig{
		// 在途达到 1000 时拒绝 → 实际最多约 999 个并发在途
		Max: 1000,
	}))

	e.POST("/login", func(c *elton.Context) (err error) {
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

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
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

出错转换处理，用于将出错转换为json或text出错响应，建议在业务逻辑中使用自定义的出错类型，使用出错中间件将相应的出错信息转换输出，可方便的汇总统计非自定义的出错类型，便于系统的优化。

**Example**
```go
package main

import (
	"errors"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
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

根据响应数据生成HTTP响应头的ETag，需要从BodyBuffer中生成，因此需要先通过Responder中间件将响应转换为Buffer或直接设置BodyBuffer。

**Example**
```go
package main

import (
	"bytes"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
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

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
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

Logger中间件，支持从请求头、响应头等获取信息，日志中标签以{}标记，支持的标签如下：

- `host` 请求的host
- `method` 请求的method
- `path` 请求的path
- `proto` 请求的协议类型
- `query` 请求的raw query
- `remote` 请求的remote addr
- `real-ip` 客户的真实IP
- `client-ip` 客户的IP，与real-ip的区别是会判断IP是否公网IP
- `scheme` HTTP或者HTTPS
- `uri` 请求的完整地址
- `referer` 请求的referer
- `userAgent` 请求的user agent
- `when` 当前时间RFC1123带时区的格式化
- `when-iso` 当前时间RFC3339的格式化
- `when-utc-iso` 当前UTC时间的ISO格式化
- `when-unix` 当前时间的unix时间戳(秒)
- `when-iso-ms` 当前时间RFC3339的格式化(毫秒)
- `when-utc-iso-ms` 当前UTC时间的ISO格式化(毫秒)
- `size` 响应数据长度(字节)
- `size-human` 响应数据长度，格式化为KB/MB(以1024换算)
- `status` 状态码
- `latency` 响应时间
- `latency-ms` 响应时间(毫秒)
- `~cookie` 表示获取cookie的值，必须以~开头，后面的表示cookie的key
- `payload-size` 提交数据长度(字节)
- `payload-size-human` 提交数据长度，格式化为KB/MB(以1024换算)
- `>header` 表示获取请求头的值，必须以>开头，后面表示header的key
- `<header` 表示获取响应头的值，必须以<开头，后面表示header的key
- `:key` 获取获取context设置的值，必须以:开头，后面表示对应的key，需要注意，设置至context中的值必须为string
- `$key` 从ENV中获取该key对应的值，必须以$开头，后面表示对应的key


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

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
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

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
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

Recover中间件，用于捕获各种panic异常，避免程序异常退出，但建议自定义recover中间件，在获取到此类异常时，发送告警后做graceful restart。 

**Example**
```go
package main

import (
	"errors"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
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

## renderer

模板渲染中间件，用于将各类模板渲染为html输出，默认支持`html`与`tmpl`两种后续文件使用`html/template`模块来渲染。

```go
package main

import (
	"errors"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewRenderer(middleware.RendererConfig{}))

	e.GET("/", func(c *elton.Context) (err error) {
		c.Body = middleware.RenderData{
			File:         "index.html",
		}
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
- `ResponderConfig.ContentType` 自定义的ContentType，默认为`application/json; charset=utf-8`

**Example**
```go
package main

import (
	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
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

## response size limiter

响应长度限制中间件，可以限制响应数据的长度，避免返回过大的数据导致网络占用过大。此中间件主要用于避免一些非法调用等导致查询过多数据。

**Example**
```go
package main

import (
	"bytes"
	"time"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewResponseSizeLimiter(middleware.ResponseSizeLimiterConfig{
		// 1MB
		MaxSize: 1024 * 1024,
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

## router concurrent limiter

按路由限制并发，避免单接口打满进程。本地实现请用 `NewLocalRouterConcurrencyLimiter`（v2 已重命名，原 `NewLocalLimiter` / `NewRCL` 见 [迁移指南](./migration-v2.md)）。

**Example**
```go
package main

import (
	"bytes"
	"time"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {
	e := elton.New()

	e.Use(middleware.NewRouterConcurrentLimiter(middleware.RouterConcurrentLimiterConfig{
		Limiter: middleware.NewLocalRouterConcurrencyLimiter(map[string]uint32{
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

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
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

静态文件中间件。常见用法：

- **OS 目录**：`middleware.NewFSStaticServe(config)`（v2 语义名；内部使用 `middleware.FS`，根目录为 `config.Path`）
- **embed**：`middleware.NewEmbedStaticServe(embedFS, config)`
- **自实现**：实现 `StaticFile`（`Exists` / `Get` / `Stat` / `NewReader` 返回 `io.ReadCloser`），再 `NewStaticServe`
- **预压缩资源**：`NewEncodingStaticServe`，按 Accept-Encoding 选择编码 FS

**Example（目录）**
```go
package main

import (
	"time"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {
	e := elton.New()

	e.GET("/*", middleware.NewFSStaticServe(middleware.StaticServeConfig{
		Path: "/tmp",
		// 客户端缓存一年
		MaxAge: 365 * 24 * time.Hour,
		// 缓存服务器缓存一个小时
		SMaxAge:             time.Hour,
		DenyQueryString:     true,
		DisableLastModified: true,
		// 无 Stat 的后端可开强 ETag
		EnableStrongETag: true,
	}))

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

**Example（go:embed）**
```go
package main

import (
	"embed"
	"time"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

//go:embed assets/*
var assets embed.FS

func main() {
	e := elton.New()

	e.GET("/static/*", middleware.NewEmbedStaticServe(assets, middleware.StaticServeConfig{
		// embed 内子路径，按实际调整；也可留空表示根
		Path:            "assets",
		MaxAge:          365 * 24 * time.Hour,
		SMaxAge:         time.Hour,
		DenyQueryString: true,
	}))

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

自定义存储（对象存储等）实现 `StaticFile` 即可：`NewReader` 须返回 `io.ReadCloser`（内存数据可用 `io.NopCloser`），关闭由框架统一负责。

## tracker

用于在客户提交类的请求添加跟踪日志，可输出query、body以及params等信息，并可设置正则匹配将关键数据加*处理。

**Example**
```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
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
## cors

跨域中间件。预检（`OPTIONS` + `Access-Control-Request-Method`）会设置 CORS 头并 `204` 短路，不调用后续中间件。

**Example**
```go
e.Use(middleware.NewCORS(middleware.CORSConfig{
	AllowOrigins:     []string{"https://app.example"},
	AllowCredentials: true,
	MaxAge:           time.Hour,
	ExposeHeaders:    []string{middleware.HeaderXRequestID},
}))
// 或任意 origin（无 credentials）：middleware.NewDefaultCORS()
```

## timeout

为请求 context 设置 deadline。业务与下游客户端应使用 `c.Context()`；超时后返回 504（`ErrRequestTimeout`）。**不会**强制中断已在运行的非协作代码。

**Example**
```go
e.Use(middleware.NewDefaultTimeout(5 * time.Second))
// 或 NewTimeout(middleware.TimeoutConfig{Timeout: 3 * time.Second})
```

## request id

确保每个请求有 ID：优先使用请求头 `X-Request-Id`（可配置），否则生成 16 字节 hex；写入 context（`middleware.GetRequestID`）、响应头，并在 `c.ID` 为空时填充。

**Example**
```go
e.Use(middleware.NewDefaultRequestID())
// logger 中可用 {>X-Request-Id} 或 {:requestId}
```
