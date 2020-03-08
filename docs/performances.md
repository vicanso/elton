---
description: 性能测试
---

`elton`的性能如何是大家都会关心的重点，下面是使用我的测试服务器(4U8线程，8G内存)的几个测试场景：


```go
package main

import (
	"bytes"

	"github.com/vicanso/elton"
)

func main() {
	d := elton.New()

	d.GET("/", func(c *elton.Context) (err error) {
		c.BodyBuffer = bytes.NewBufferString("Hello, World!")
		return
	})
	err := d.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

```bash
wrk -c 1000 -t 10 --latency 'http://127.0.0.1:3000/'
Running 10s test @ http://127.0.0.1:3000/
  10 threads and 1000 connections

  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    14.88ms   23.23ms 391.57ms   89.74%
    Req/Sec    12.05k     3.39k   48.51k    74.57%
  Latency Distribution
     50%    7.80ms
     75%   14.15ms
     90%   39.04ms
     99%  119.70ms
  1203441 requests in 10.10s, 146.90MB read
Requests/sec: 119204.55
Transfer/sec:     14.55MB
```

从上面的测试可以看出，每秒可以处理100K的请求数，这看着性能是好高，但实际上这种测试的意义不太大，不过总可以让大家放心不至于拖后腿。

`elton`的亮点是在响应数据中间件的处理，以简单的方式返回正常或出错的响应数据，下面我们来测试一下这两种场景的性能表现。


```go
package main

import (
	"strings"

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"

	errorhandler "github.com/vicanso/elton-error-handler"
	responder "github.com/vicanso/elton-responder"
)

type (
	HelloWord struct {
		Content string  `json:"content,omitempty"`
		Size    int     `json:"size,omitempty"`
		Price   float32 `json:"price,omitempty"`
		VIP     bool    `json:"vip,omitempty"`
	}
)

func main() {
	d := elton.New()

	arr := make([]string, 0)
	for i := 0; i < 100; i++ {
		arr = append(arr, "花褪残红青杏小。燕子飞时，绿水人家绕。枝上柳绵吹又少，天涯何处无芳草！")
	}
	content := strings.Join(arr, "\n")

	d.Use(errorhandler.NewDefault())
	d.Use(responder.NewDefault())

	d.GET("/", func(c *elton.Context) (err error) {
		c.Body = &HelloWord{
			Content: content,
			Size:    100,
			Price:   10.12,
			VIP:     true,
		}
		return
	})

	d.GET("/error", func(c *elton.Context) (err error) {
		err = hes.New("abcd")
		return
	})
	err := d.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}

```

```bash
wrk -c 1000 -t 10 --latency 'http://127.0.0.1:3000/'
Running 10s test @ http://127.0.0.1:3000/
  10 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    64.97ms   91.92ms 882.31ms   86.18%
    Req/Sec     3.83k     1.16k   28.47k    82.28%
  Latency Distribution
     50%   25.92ms
     75%   99.57ms
     90%  189.24ms
     99%  402.33ms
  380724 requests in 10.10s, 3.86GB read
Requests/sec:  37699.90
Transfer/sec:    391.61MB
```

```bash
wrk -c 1000 -t 10 --latency 'http://127.0.0.1:7001/error'
Running 10s test @ http://127.0.0.1:7001/error
  10 threads and 1000 connections


  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    15.18ms   21.10ms 319.94ms   89.10%
    Req/Sec    10.46k     2.81k   24.50k    67.50%
  Latency Distribution
     50%    8.68ms
     75%   17.51ms
     90%   38.51ms
     99%  103.63ms
  1041735 requests in 10.10s, 190.75MB read
  Non-2xx or 3xx responses: 1041735
Requests/sec: 103142.00
Transfer/sec:     18.89MB
```

对于正常返回（数据量为10KB）的struct做序列化时，性能会有所降低，从测试结果可以看出，每秒还是可以处理37K的请求，出错的转换处理效率更高，每秒能处理103K的请求。



下面是`gin`的测试结果：

```go
package main

import (
	"strings"

	"github.com/gin-gonic/gin"
)

type (
	HelloWord struct {
		Content string  `json:"content,omitempty"`
		Size    int     `json:"size,omitempty"`
		Price   float32 `json:"price,omitempty"`
		VIP     bool    `json:"vip,omitempty"`
	}
)

func main() {

	arr := make([]string, 0)
	for i := 0; i < 100; i++ {
		arr = append(arr, "花褪残红青杏小。燕子飞时，绿水人家绕。枝上柳绵吹又少，天涯何处无芳草！")
	}
	content := strings.Join(arr, "\n")

	router := gin.New()

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, &HelloWord{
			Content: content,
			Size:    100,
			Price:   10.12,
			VIP:     true,
		})
	})
	router.Run(":7001")
}
```

```bash
wrk -c 1000 -t 10 --latency 'http://127.0.0.1:7001/'
Running 10s test @ http://127.0.0.1:7001/
  10 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    61.18ms   83.74ms 809.55ms   85.47%
    Req/Sec     3.85k     0.88k    6.98k    69.20%
  Latency Distribution
     50%   25.46ms
     75%   96.02ms
     90%  178.73ms
     99%  356.19ms
  383967 requests in 10.04s, 3.89GB read
Requests/sec:  38254.94
Transfer/sec:    397.37MB
```

从上面的测试数据可以看出，因为都是基于[httprouter](https://github.com/julienschmidt/httprouter)，`elton`的性能与`gin`整体上基本一致，无需太过担忧性能问题。