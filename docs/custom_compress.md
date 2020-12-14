---
description: 自定义压缩
---

绝大多数客户端都支持多种压缩方式，需要不同的场景选择适合的压缩算法，微服务之间的调用更是可以选择一些高性能的压缩方式，下面来介绍如何编写自定义的压缩中间件，主要的要点如下：

- 根据请求头`Accept-Encoding`判断客户端支持的压缩算法
- 设定最小压缩长度，避免对较小数据的压缩浪费性能
- 根据响应头`Content-Type`判断只压缩文本类的响应数据
- 根据场景平衡压缩率与性能的选择，如内网的可以选择snappy，lz4等高效压缩算法

[elton-compress](https://github.com/vicanso/elton-compress)中间件提供了其它几种常用的压缩方式，包括`brotli`以及`snappy`等。如果要增加压缩方式，只需要实现`Compressor`的三个函数则可。

```go
// Compressor compressor interface
Compressor interface {
	// Accept accept check function
	Accept(c *elton.Context, bodySize int) (acceptable bool, encoding string)
	// Compress compress function
	Compress([]byte) (*bytes.Buffer, error)
	// Pipe pipe function
	Pipe(*elton.Context) error
}
```

下面是elton-compress中间件lz4压缩的实现代码：

```go
package compress

import (
	"bytes"
	"io"

	"github.com/pierrec/lz4"
	"github.com/vicanso/elton"
)

const (
	// Lz4Encoding lz4 encoding
	Lz4Encoding = "lz4"
)

type (
	// Lz4Compressor lz4 compress
	Lz4Compressor struct {
		Level     int
		MinLength int
	}
)

func (l *Lz4Compressor) getMinLength() int {
	if l.MinLength == 0 {
		return defaultCompressMinLength
	}
	return l.MinLength
}

// Accept check accept encoding
func (l *Lz4Compressor) Accept(c *elton.Context, bodySize int) (acceptable bool, encoding string) {
	// 如果数据少于最低压缩长度，则不压缩
	if bodySize >= 0 && bodySize < l.getMinLength() {
		return
	}
	return AcceptEncoding(c, Lz4Encoding)
}

// Compress lz4 compress
func (l *Lz4Compressor) Compress(buf []byte) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	w := lz4.NewWriter(buffer)
	defer w.Close()
	w.Header.CompressionLevel = l.Level
	_, err := w.Write(buf)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

// Pipe lz4 pipe compress
func (l *Lz4Compressor) Pipe(c *elton.Context) (err error) {
	r := c.Body.(io.Reader)
	closer, ok := c.Body.(io.Closer)
	if ok {
		defer closer.Close()
	}
	w := lz4.NewWriter(c.Response)
	w.Header.CompressionLevel = l.Level
	defer w.Close()
	_, err = io.Copy(w, r)
	return
}
```

下面调用示例：

```go
package main

import (
	"bytes"

	"github.com/vicanso/elton"
	compress "github.com/vicanso/elton-compress"
)

func main() {
	d := elton.New()

	conf := compress.Config{}
	lz4 := &compress.Lz4Compressor{
		Level:     2,
		MinLength: 1024,
	}
	conf.AddCompressor(lz4)
	d.Use(compress.New(conf))

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


```
curl -H 'Accept-Encoding:lz4' -v 'http://127.0.0.1:3000'
* Rebuilt URL to: http://127.0.0.1:3000/
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to 127.0.0.1 (127.0.0.1) port 3000 (#0)
> GET / HTTP/1.1
> Host: 127.0.0.1:3000
> User-Agent: curl/7.54.0
> Accept: */*
> Accept-Encoding:lz4
>
< HTTP/1.1 200 OK
< Content-Encoding: lz4
< Content-Length: 103
< Content-Type: text/plain; charset=utf-8
< Vary: Accept-Encoding
...
```

从响应头中可以看出，数据已经压缩为`lz4`的格式，数据长度仅为`103`字节，节约了带宽。
