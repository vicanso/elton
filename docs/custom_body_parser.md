---
description: 自定义Body Parser
---

elton-body-parser只提供对`application/json`以及`application/x-www-form-urlencoded`转换为json字节的处理，在实际使用中还存在一些其它的场景。如`xml`，自定义数据结构等。

在实际项目中，统计数据我一般记录至influxdb，为了性能的考虑，统计数据是批量提交（如每1000个统计点提交一次）。数据提交的时候，重复的字符比较多，为了减少带宽的占用，所以先做压缩处理。考虑到性能的原因，采用了`snappy`压缩处理。下面是抽取出来的示例代码：

```go
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/snappy"
	"github.com/vicanso/elton"
)

// 仅示例，对于出错直接panic
func post() {
	// weather,location=us-midwest temperature=82 1465839830100400200
	max := 1000
	arr := make([]string, max)
	for i := 0; i < max; i++ {
		arr[i] = "weather,location=us-midwest temperature=82 " + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	var dst []byte
	data := snappy.Encode(dst, []byte(strings.Join(arr, "\n")))

	req, err := http.NewRequest("POST", "http://127.0.0.1:3000/influx", bytes.NewReader(data))
	req.Header.Set(elton.HeaderContentType, ContentTypeIfx)
	if err != nil {
		panic(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	result, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(result))
}

const (
	// ContentTypeIfx influx data type
	ContentTypeIfx = "application/ifx"
)

// NewInfluxParser influx parser
func NewInfluxParser() elton.Handler {
	return func(c *elton.Context) (err error) {
		// 对于非POST请求，以及数据类型不匹配的，则跳过
		if c.Request.Method != http.MethodPost ||
			c.GetRequestHeader(elton.HeaderContentType) != ContentTypeIfx {
			return c.Next()
		}
		body, err := ioutil.ReadAll(c.Request.Body)
		// 如果读取数据时出错，直接返回
		if err != nil {
			return
		}
		var dst []byte
		data, err := snappy.Decode(dst, body)
		// 如果解压出错，直接返回（也可再自定义出错类型，方便排查）
		if err != nil {
			return
		}
		// 至此则解压生成提交的数据了
		c.RequestBody = data
		return c.Next()
	}
}

func main() {
	e := elton.New()
	go func() {
		// 等待一秒让elton启动（仅为了测试方便，直接客户端服务端同一份代码）
		time.Sleep(time.Second)
		post()
	}()

	e.Use(NewInfluxParser())

	e.POST("/influx", func(c *elton.Context) (err error) {
		points := strings.SplitN(string(c.RequestBody), "\n", -1)
		c.BodyBuffer = bytes.NewBufferString("add " + strconv.Itoa(len(points)) + " points to influxdb done")
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```

通过各类自定义的中间件，可以实现各种不同的提交数据的解析，只要将解析结果保存至`Context.RequestBody`中，后续则由处理函数再将字节转换为相对应的结构，简单易用。

[elton-body-parser](https://github.com/vicanso/elton-body-parser)提供自定义Decoder方式，可以按实际使用添加Decoder，上面的实现可以简化为：

```go
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/snappy"
	"github.com/vicanso/elton"
	bodyparser "github.com/vicanso/elton-body-parser"
)

// 仅示例，对于出错直接panic
func post() {
	// weather,location=us-midwest temperature=82 1465839830100400200
	max := 1000
	arr := make([]string, max)
	for i := 0; i < max; i++ {
		arr[i] = "weather,location=us-midwest temperature=82 " + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	var dst []byte
	data := snappy.Encode(dst, []byte(strings.Join(arr, "\n")))

	req, err := http.NewRequest("POST", "http://127.0.0.1:3000/influx", bytes.NewReader(data))
	req.Header.Set(elton.HeaderContentType, ContentTypeIfx)
	if err != nil {
		panic(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	result, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(result))
}

const (
	// ContentTypeIfx influx data type
	ContentTypeIfx = "application/ifx"
)

func main() {
	e := elton.New()
	go func() {
		// 等待一秒让elton启动（仅为了测试方便，直接客户端服务端同一份代码）
		time.Sleep(time.Second)
		post()
	}()

	conf := bodyparser.Config{
		// 设置对哪些content type处理，默认只处理application/json
		ContentTypeValidate: func(c *elton.Context) bool {
			ct := c.GetRequestHeader(elton.HeaderContentType)
			return regexp.MustCompile("application/json|" + ContentTypeIfx).MatchString(ct)
		},
	}
	// gzip解压
	conf.AddDecoder(bodyparser.NewGzipDecoder())
	// json decoder
	conf.AddDecoder(bodyparser.NewJSONDecoder())
	// 添加自定义influx的decoder
	conf.AddDecoder(&bodyparser.Decoder{
		// 判断是否符合该decoder
		Validate: func(c *elton.Context) bool {
			return c.GetRequestHeader(elton.HeaderContentType) == ContentTypeIfx
		},
		// 解压snappy
		Decode: func(c *elton.Context, orginalData []byte) (data []byte, err error) {
			var dst []byte
			data, err = snappy.Decode(dst, orginalData)
			return
		},
	})

	e.Use(bodyparser.New(conf))

	e.POST("/influx", func(c *elton.Context) (err error) {
		points := strings.SplitN(string(c.RequestBody), "\n", -1)
		c.BodyBuffer = bytes.NewBufferString("add " + strconv.Itoa(len(points)) + " points to influxdb done")
		return
	})

	err := e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```