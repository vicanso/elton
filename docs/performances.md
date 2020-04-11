---
description: 性能测试
---

`elton`的性能如何是大家都会关心的重点，下面是使用测试服务器(4U8线程，8G内存)的几个测试场景，go版本为1.14：

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
    Latency    11.24ms   12.07ms 127.54ms   87.12%
    Req/Sec    11.39k     2.60k   33.20k    74.42%
  Latency Distribution
     50%    7.24ms
     75%   13.95ms
     90%   26.98ms
     99%   56.30ms
  1129086 requests in 10.09s, 139.98MB read
Requests/sec: 111881.19
Transfer/sec:     13.87MB
```

从上面的测试可以看出，每秒可以处理110K的请求数，这看着性能是好高，但实际上这种测试的意义不太大，不过总可以让大家放心不至于拖后腿。

`elton`的亮点是在响应数据中间件的处理，以简单的方式返回正常或出错的响应数据，下面我们来测试一下这两种场景的性能表现。


```go
package main

import (
	"strings"

	"github.com/vicanso/elton"
	"github.com/vicanso/elton/middleware"
	"github.com/vicanso/hes"
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

	d.Use(middleware.NewDefaultError())
	d.Use(middleware.NewDefaultResponder())

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
    Latency    46.41ms   58.15ms 606.12ms   83.56%
    Req/Sec     4.22k   798.75     7.18k    69.90%
  Latency Distribution
     50%   15.31ms
     75%   79.23ms
     90%  129.41ms
     99%  240.98ms
  420454 requests in 10.07s, 4.26GB read
Requests/sec:  41734.70
Transfer/sec:    432.80MB
```

```bash
wrk -c 1000 -t 10 --latency 'http://127.0.0.1:3000/error'
Running 10s test @ http://127.0.0.1:3000/error
  10 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    11.29ms   11.36ms 146.95ms   86.59%
    Req/Sec    10.91k     2.37k   21.86k    70.08%
  Latency Distribution
     50%    7.62ms
     75%   14.23ms
     90%   26.56ms
     99%   53.32ms
  1083752 requests in 10.10s, 142.63MB read
  Non-2xx or 3xx responses: 1083752
Requests/sec: 107344.19
Transfer/sec:     14.13MB
```

对于正常返回（数据量为10KB）的struct做序列化时，性能会有所降低，从测试结果可以看出，每秒还是可以处理41K的请求，出错的转换处理效率更高，每秒能处理107K的请求。



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
	router.Run(":3000")
}
```

```bash
wrk -c 1000 -t 10 --latency 'http://127.0.0.1:3000/'
Running 10s test @ http://127.0.0.1:3000/
  10 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    52.17ms   66.58ms 629.46ms   83.46%
    Req/Sec     3.97k     0.91k   13.91k    72.16%
  Latency Distribution
     50%   16.37ms
     75%   89.93ms
     90%  145.96ms
     99%  277.75ms
  394628 requests in 10.10s, 4.00GB read
Requests/sec:  39075.49
Transfer/sec:    405.90MB
```

从上面的测试数据可以看出`elton`的性能与`gin`整体上基本一致，无需太过担忧性能问题，需要注意的是elton不再使用httprouter来处理路由，路由参数支持更多自定义的处理，因此路由的查询性能有所下降。