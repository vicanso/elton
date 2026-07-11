---
description: 自定义压缩
---

绝大多数客户端都支持多种压缩方式，需要按场景选择合适算法：公网偏重压缩率（br/zstd/gzip），内网微服务可偏向低 CPU（snappy/lz4 等）。

## 内置压缩（2.0）

`github.com/vicanso/elton/v2/middleware` **已内置**：

| 构造函数 | 编码 |
|----------|------|
| `NewGzipCompressor()` | gzip |
| `NewBrCompressor()` | br |
| `NewZstdCompressor()` | zstd |

```go
e.Use(middleware.NewCompress(middleware.NewCompressConfig(
	middleware.NewGzipCompressor(),
	middleware.NewBrCompressor(),
	middleware.NewZstdCompressor(),
)))
```

`NewDefaultCompress()` 仅启用 gzip。缓存侧还有 `NewCacheGzipCompressor` / `NewCacheBrCompressor` / `NewCacheZstdCompressor` 等，与 HTTP cache 中间件配合使用。

## 扩展自定义算法

实现 `Compressor` 三个方法即可接入任意编码：

```go
// Compressor compressor interface
type Compressor interface {
	// Accept 是否接受该编码（通常看 Accept-Encoding 与 body 大小）
	Accept(c *elton.Context, bodySize int) (acceptable bool, encoding string)
	// Compress 缓冲压缩；level 由中间件 DynamicLevel 等传入，可忽略
	Compress([]byte, ...int) (*bytes.Buffer, error)
	// Pipe 流式 Body（io.Reader）压缩写出
	Pipe(*elton.Context) error
}
```

常见注意点：

- 根据请求头 `Accept-Encoding` 判断客户端是否支持
- 设定最小压缩长度，避免对过小 body 浪费 CPU
- 根据响应头 `Content-Type` 只压缩文本类（中间件默认 checker 为 `text|javascript|json|wasm|font`）
- 内网可选用 snappy、lz4 等高效算法（需自实现或第三方库）

下面以 **lz4** 为例（算法本身不在 elton 仓库内，依赖 `github.com/pierrec/lz4`）：

```go
package compress

import (
	"bytes"
	"io"

	"github.com/pierrec/lz4"
	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

const Lz4Encoding = "lz4"

type Lz4Compressor struct {
	Level     int
	MinLength int
}

func (l *Lz4Compressor) getMinLength() int {
	if l.MinLength == 0 {
		return middleware.DefaultCompressMinLength
	}
	return l.MinLength
}

func (l *Lz4Compressor) Accept(c *elton.Context, bodySize int) (bool, string) {
	if bodySize >= 0 && bodySize < l.getMinLength() {
		return false, ""
	}
	return middleware.AcceptEncoding(c, Lz4Encoding)
}

func (l *Lz4Compressor) Compress(buf []byte, levels ...int) (*bytes.Buffer, error) {
	level := l.Level
	if len(levels) > 0 {
		level = levels[0]
	}
	buffer := new(bytes.Buffer)
	w := lz4.NewWriter(buffer)
	if level != 0 {
		w.Header.CompressionLevel = level
	}
	_, err := w.Write(buf)
	if err != nil {
		return nil, err
	}
	if err = w.Close(); err != nil {
		return nil, err
	}
	return buffer, nil
}

func (l *Lz4Compressor) Pipe(c *elton.Context) error {
	r, ok := c.Body.(io.Reader)
	if !ok {
		return nil
	}
	closer, ok := r.(io.Closer)
	if ok {
		defer func() {
			_ = closer.Close()
		}()
	}
	w := lz4.NewWriter(c.Response)
	if l.Level != 0 {
		w.Header.CompressionLevel = l.Level
	}
	defer func() {
		_ = w.Close()
	}()
	_, err := io.Copy(w, r)
	return err
}
```

调用示例：

```go
package main

import (
	"bytes"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
	// 将上面的 Lz4Compressor 放在本模块或独立包中
)

func main() {
	d := elton.New()

	conf := middleware.NewCompressConfig(
		middleware.NewGzipCompressor(),
		&Lz4Compressor{
			Level:     2,
			MinLength: 1024,
		},
	)
	d.Use(middleware.NewCompress(conf))

	d.GET("/", func(c *elton.Context) (err error) {
		b := new(bytes.Buffer)
		for i := 0; i < 1000; i++ {
			b.WriteString("Hello, World!\n")
		}
		c.SetHeader(elton.HeaderContentType, "text/plain; charset=utf-8")
		c.BodyBuffer = b
		return
	})

	err := d.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

```bash
curl -H 'Accept-Encoding: lz4' -v 'http://127.0.0.1:3000'
```
