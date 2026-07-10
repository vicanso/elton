// MIT License

// Copyright (c) 2020 Tree Xie

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package middleware

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/vicanso/elton/v2"
)

const (
	host             = "host"
	method           = "method"
	path             = "path"
	proto            = "proto"
	query            = "query"
	remote           = "remote"
	realIP           = "real-ip"
	clientIP         = "client-ip"
	scheme           = "scheme"
	uri              = "uri"
	referer          = "referer"
	userAgent        = "userAgent"
	when             = "when"
	whenISO          = "when-iso"
	whenUTCISO       = "when-utc-iso"
	whenUnix         = "when-unix"
	whenISOMs        = "when-iso-ms"
	whenUTCISOMs     = "when-utc-iso-ms"
	size             = "size"
	sizeHuman        = "size-human"
	status           = "status"
	latency          = "latency"
	latencyMs        = "latency-ms"
	cookie           = "cookie"
	payloadSize      = "payload-size"
	payloadSizeHuman = "payload-size-human"
	requestHeader    = "requestHeader"
	responseHeader   = "responseHeader"
	ctx              = "context"
	httpProto        = "HTTP"
	httpsProto       = "HTTPS"
	// get from env
	env = "env"

	kbytes = 1024
	mbytes = 1024 * 1024

	// LoggerCommon combined log format
	LoggerCombined = `{remote} {when-iso} "{method} {uri} {proto}" {status} {size-human} "{referer}" "{userAgent}"`
	// LoggerCommon common log format
	LoggerCommon = `{remote} {when-iso} "{method} {uri} {proto}" {status} {size-human}`
	// LoggerShort short log format
	LoggerShort = `{remote} {method} {uri} {proto} {status} {size-human} - {latency-ms} ms`
	// LoggerTiny tiny log format
	LoggerTiny = `{method} {uri} {status} {size-human} - {latency-ms} ms`
)

type (
	// LoggerTag logger tag
	LoggerTag struct {
		category string
		data     string
	}
	// OnLog on log function
	OnLog func(string, *elton.Context)
	// LoggerConfig logger config
	LoggerConfig struct {
		// DefaultFill default fill for empty value
		DefaultFill string
		Format      string
		OnLog       OnLog
		Skipper     elton.Skipper
	}
)

// byteSliceToString converts a []byte to string without a heap allocation.
func byteSliceToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
func cutLog(str string) string {
	l := len(str)
	if l == 0 {
		return str
	}
	ch := str[l-1]
	if ch == '0' || ch == '.' {
		return cutLog(str[0 : l-1])
	}
	return str
}

// getHumanReadableSize get the size for human
func getHumanReadableSize(size int) string {
	if size < kbytes {
		return fmt.Sprintf("%dB", size)
	}
	fSize := float64(size)
	if size < mbytes {
		s := cutLog(fmt.Sprintf("%.2f", (fSize / kbytes)))
		return s + "KB"
	}
	s := cutLog(fmt.Sprintf("%.2f", (fSize / mbytes)))
	return s + "MB"
}

// getTimeConsuming 获取使用耗时(ms)
func getTimeConsuming(startedAt time.Time) int {
	v := startedAt.UnixNano()
	now := time.Now().UnixNano()
	return int((now - v) / 1000000)
}

// parseLoggerTags 将日志格式字符串解析为tag列表。
// 格式中 {xxx} 为动态tag，其余字面文本解析为fill tag原样输出。
// tag支持以下前缀迷你语法：
//   - ~name  : 请求cookie的值，如 {~sid}
//   - >name  : 请求头的值，如 {>X-Request-Id}
//   - <name  : 响应头的值，如 {<X-Response-Id}
//   - :name  : context store中的值（c.Set保存的字符串），如 {:userId}
//   - $name  : 环境变量的值，如 {$HOSTNAME}
//   - 其它   : 内置tag，全集见 getLoggerTagValue：
//     host method path proto query remote real-ip client-ip scheme uri
//     referer userAgent when when-iso when-utc-iso when-unix when-iso-ms
//     when-utc-iso-ms size size-human status latency latency-ms cookie
//     payload-size payload-size-human
func parseLoggerTags(desc []byte) []*LoggerTag {
	reg := regexp.MustCompile(`\{[\S]+?\}`)

	index := 0
	arr := make([]*LoggerTag, 0)
	fillCategory := "fill"
	for {
		result := reg.FindIndex(desc[index:])
		if result == nil {
			break
		}
		start := index + result[0]
		end := index + result[1]
		if start != index {
			arr = append(arr, &LoggerTag{
				category: fillCategory,
				data:     byteSliceToString(desc[index:start]),
			})
		}
		k := desc[start+1 : end-1]
		switch k[0] {
		case byte('~'):
			arr = append(arr, &LoggerTag{
				category: cookie,
				data:     byteSliceToString(k[1:]),
			})
		case byte('>'):
			arr = append(arr, &LoggerTag{
				category: requestHeader,
				data:     byteSliceToString(k[1:]),
			})
		case byte('<'):
			arr = append(arr, &LoggerTag{
				category: responseHeader,
				data:     byteSliceToString(k[1:]),
			})
		case byte(':'):
			arr = append(arr, &LoggerTag{
				category: ctx,
				data:     byteSliceToString(k[1:]),
			})
		case byte('$'):
			arr = append(arr, &LoggerTag{
				category: env,
				data:     os.Getenv(string(k[1:])),
			})
		default:
			arr = append(arr, &LoggerTag{
				category: byteSliceToString(k),
				data:     "",
			})
		}
		index = result[1] + index
	}
	if index < len(desc) {
		arr = append(arr, &LoggerTag{
			category: fillCategory,
			data:     byteSliceToString(desc[index:]),
		})
	}
	return arr
}

// getLoggerTagValue 获取logger tag对应的值
func getLoggerTagValue(c *elton.Context, tag *LoggerTag, startedAt time.Time) string {
	switch tag.category {
	case host:
		return c.Request.Host
	case method:
		return c.Request.Method
	case path:
		p := c.Request.URL.Path
		if p == "" {
			p = "/"
		}
		return p
	case proto:
		return c.Request.Proto
	case query:
		return c.Request.URL.RawQuery
	case remote:
		return c.Request.RemoteAddr
	case realIP:
		return c.RealIP()
	case clientIP:
		return c.ClientIP()
	case scheme:
		if c.Request.TLS != nil {
			return httpsProto
		}
		return httpProto
	case uri:
		return c.Request.RequestURI
	case cookie:
		cookie, err := c.Cookie(tag.data)
		if err != nil {
			return ""
		}
		return cookie.Value
	case requestHeader:
		return c.Request.Header.Get(tag.data)
	case responseHeader:
		return c.GetHeader(tag.data)
	case ctx:
		return elton.GetContextValue[string](c, tag.data)
	case referer:
		return c.Request.Referer()
	case userAgent:
		return c.Request.UserAgent()
	case when:
		return time.Now().Format(time.RFC1123Z)
	case whenISO:
		return time.Now().Format(time.RFC3339)
	case whenUTCISO:
		return time.Now().UTC().Format("2006-01-02T15:04:05Z")
	case whenISOMs:
		return time.Now().Format("2006-01-02T15:04:05.999Z07:00")
	case whenUTCISOMs:
		return time.Now().UTC().Format("2006-01-02T15:04:05.999Z")
	case whenUnix:
		return strconv.FormatInt(time.Now().Unix(), 10)
	case status:
		statusCode := c.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		return strconv.Itoa(statusCode)
	case payloadSize:
		return strconv.Itoa(len(c.RequestBody))
	case payloadSizeHuman:
		return getHumanReadableSize(len(c.RequestBody))
	case size:
		if c.IsReaderBody() {
			return "-"
		}
		bodyBuf := c.BodyBuffer
		if bodyBuf == nil {
			return "0"
		}
		return strconv.Itoa(bodyBuf.Len())
	case sizeHuman:
		if c.IsReaderBody() {
			return "-"
		}
		bodyBuf := c.BodyBuffer
		if bodyBuf == nil {
			return "0B"
		}
		return getHumanReadableSize(bodyBuf.Len())
	case latency:
		return time.Since(startedAt).String()
	case latencyMs:
		ms := getTimeConsuming(startedAt)
		return strconv.Itoa(ms)
	default:
		return tag.data
	}
}

// formatLog 格式化访问日志信息
func formatLog(c *elton.Context, tags []*LoggerTag, startedAt time.Time, defaultFill string) string {
	sb := strings.Builder{}
	// 预估每个tag的值为16字节
	sb.Grow(len(tags) * 16)
	for _, tag := range tags {
		v := getLoggerTagValue(c, tag, startedAt)
		if v == "" {
			v = defaultFill
		}
		sb.WriteString(v)
	}
	return sb.String()
}

// NewLogger returns a new logger middleware, it can log the field of query string, header, cookie and context.
// Format的tag语法见 parseLoggerTags，预置格式有 LoggerCombined/LoggerCommon/LoggerShort/LoggerTiny。
// It will throw a panic if the Format is empty.
// It will throw a panic if the OnLog function is nil.
func NewLogger(config LoggerConfig) elton.Handler {
	if config.Format == "" {
		panic("logger require format")
	}
	if config.OnLog == nil {
		panic("logger require on log function")
	}
	tags := parseLoggerTags([]byte(config.Format))
	skipper := getSkipper(config.Skipper)
	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		startedAt := time.Now()
		err := c.Next()
		str := formatLog(c, tags, startedAt, config.DefaultFill)
		config.OnLog(str, c)
		return err
	}
}
