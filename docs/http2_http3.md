---
description: HTTP2与HTTP3的支持
---

绝大部分的浏览器都已支持http2，后端支持http2仅需要更新nginx版本，调整配置则可。golang的http模块已支持http2的处理，下面主要介绍如何使用Elton支持http2。

## 生成证书

浏览器只支持通过https方式使用http2，为了开发方便，可以生成开发环境使用的证书，[mkcert](https://github.com/FiloSottile/mkcert)仅需几条命令则可生成多域名证书。

```bash
// 生成证书
mkcert me.dev localhost 127.0.0.1 ::1
// 安装证书
mkcert -install
```

## 启动服务

因为golang本身http模块已支持http2，则只需要以https的方式启动服务则可。

```go
package main

import (
	"bytes"

	"github.com/vicanso/elton"
)

func main() {
	e := elton.New()
	e.GET("/", func(c *elton.Context) error {
		c.BodyBuffer = bytes.NewBufferString("Hello, World!")
		return nil
	})

	certFile := "/tmp/me.dev+3.pem"
	keyFile := "/tmp/me.dev+3-key.pem"
	err := e.ListenAndServeTLS(":3000", certFile, keyFile)
	if err != nil {
		panic(err)
	}
}
```

上面例子中证书是以文件的形式保存，实际使用时证书都统一存储，加密访问（如保存在数据库等），下面的例子讲解如果使用[]byte来初始化TLS：

```go
package main

import (
	"bytes"
	"crypto/tls"
	"io"
	"os"

	"github.com/vicanso/elton"
)

// 获取证书内容
func getCert() (cert []byte, key []byte, err error) {
	// 此处仅简单从文件中读取，在实际使用，是从数据库中读取
	cert, err = os.ReadFile("/tmp/me.dev+3.pem")
	if err != nil {
		return
	}
	key, err = os.ReadFile("/tmp/me.dev+3-key.pem")
	if err != nil {
		return
	}
	return
}

func main() {
	e := elton.New()
	cert, key, err := getCert()
	if err != nil {
		panic(err)
	}
	// 先初始化tls配置，生成证书
	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = make([]tls.Certificate, 1)
	tlsConfig.Certificates[0], err = tls.X509KeyPair(cert, key)
	if err != nil {
		panic(err)
	}
	e.Server.TLSConfig = tlsConfig

	e.GET("/", func(c *elton.Context) error {
		c.BodyBuffer = bytes.NewBufferString("hello world!")
		return nil
	})

	err = e.ListenAndServeTLS(":3000", "", "")
	if err != nil {
		panic(err)
	}
}
```

## h2c

golang默认的HTTP2需要在以https的方式提供，对于内部系统间的调用，如果希望以http的方式使用http2，那么可以考虑h2c的处理。下面的代码示例包括了服务端与客户端怎么使用以http的方式使用http2。

```go
package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var http2Client = &http.Client{
	// 强制使用http2
	Transport: &http2.Transport{
		// 允许使用http的方式
		AllowHTTP: true,
		// tls的dial覆盖
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
	},
}

func main() {
	go func() {
		time.Sleep(time.Second)
		resp, err := http2Client.Get("http://127.0.0.1:3000/")
		if err != nil {
			panic(err)
		}
		fmt.Println(resp.Proto)
	}()

	e := elton.New()

	e.Use(middleware.NewDefaultResponder())

	e.GET("/", func(c *elton.Context) error {
		c.Body = "Hello, World!"
		return nil
	})
	// http1与http2均支持
	e.Server = &http.Server{
		Handler: h2c.NewHandler(e, &http2.Server{}),
	}

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

```
curl -v --http2-prior-knowledge http://127.0.0.1:3000/
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to 127.0.0.1 (127.0.0.1) port 3000 (#0)
* Using HTTP2, server supports multi-use
* Connection state changed (HTTP/2 confirmed)
* Copying HTTP/2 data in stream buffer to connection buffer after upgrade: len=0
* Using Stream ID: 1 (easy handle 0x7f8d9a804600)
> GET / HTTP/2
> Host: 127.0.0.1:3000
> User-Agent: curl/7.54.0
> Accept: */*
>
* Connection state changed (MAX_CONCURRENT_STREAMS updated)!
< HTTP/2 200
```

## http3

http3现在支持的浏览器只有chrome canary以及firefox最新版本，虽然http3的标准方案已确定，但是需要注意http3模块的使用范围并不广泛，建议不要在正式环境中大规模使用。下面是使用[quic-go](https://github.com/lucas-clemente/quic-go)使用http3的示例：

```go
package main

import (
	"bytes"
	"crypto/tls"
	"log"
	"net/http"
	"time"

	"github.com/lucas-clemente/quic-go/http3"
	"github.com/vicanso/elton"
)

const listenAddr = ":4000"

// 获取证书内容
func getCert() (cert []byte, key []byte, err error) {
	// 此处仅简单从文件中读取，在实际使用，是从数据库中读取
	cert, err = os.ReadFile("/tmp/me.dev+3.pem")
	if err != nil {
		return
	}
	key, err = os.ReadFile("/tmp/me.dev+3-key.pem")
	if err != nil {
		return
	}
	return
}

func http3Get() {
	client := &http.Client{
		Transport: &http3.RoundTripper{},
	}
	resp, err := client.Get("https://127.0.0.1" + listenAddr + "/")
	if err != nil {
		log.Fatalln("http3 get fail ", err)
		return
	}
	log.Println("http3 get success", resp.Proto, resp.Status, resp.Header)
}

func main() {
	// 延时一秒后以http3的访问访问
	go func() {
		time.Sleep(time.Second)
		http3Get()
	}()

	e := elton.New()

	cert, key, err := getCert()
	if err != nil {
		panic(err)
	}
	// 先初始化tls配置，生成证书
	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = make([]tls.Certificate, 1)
	tlsConfig.Certificates[0], err = tls.X509KeyPair(cert, key)
	if err != nil {
		panic(err)
	}
	e.Server.TLSConfig = tlsConfig.Clone()

	// 初始化http3服务
	http3Server := http3.Server{
		Server: &http.Server{
			Handler: e,
			Addr:    listenAddr,
		},
	}
	http3Server.TLSConfig = tlsConfig.Clone()

	e.Use(func(c *elton.Context) error {
		http3Server.SetQuicHeaders(c.Header())
		return c.Next()
	})

	e.GET("/", func(c *elton.Context) error {
		c.BodyBuffer = bytes.NewBufferString("hello " + c.Request.Proto + "!")
		return nil
	})

	go func() {
		// http3
		err := http3Server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	// https
	err = e.ListenAndServeTLS(listenAddr, "", "")
	if err != nil {
		panic(err)
	}
}
```

curl可以自己编译支持http3的版本，编译方法可参考[curl http3](https://github.com/curl/curl/blob/master/docs/HTTP3.md)，下面是使用curl的调用示例：

```
curl3 --http3 'https://me.dev:4000/' -v
*   Trying 127.0.0.1:4000...
* Sent QUIC client Initial, ALPN: h3-24h3-23
* h3 [:method: GET]
* h3 [:path: /]
* h3 [:scheme: https]
* h3 [:authority: me.dev:4000]
* h3 [user-agent: curl/7.68.0-DEV]
* h3 [accept: */*]
* Using HTTP/3 Stream ID: 0 (easy handle 0x7fad4f011e00)
> GET / HTTP/3
> Host: me.dev:4000
> user-agent: curl/7.68.0-DEV
> accept: */*
>
< HTTP/3 200
< content-length: 13
< alt-srv: h3-24=":4000"; ma=86400
<
* Connection #0 to host me.dev left intact
hello HTTP/3!
```