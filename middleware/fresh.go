package middleware

import (
	"net/http"

	"github.com/vicanso/cod"
	"github.com/vicanso/fresh"
)

type (
	// FreshConfig fresh config
	FreshConfig struct {
		Skipper Skipper
	}
)

// NewFresh create a fresh checker
func NewFresh(config FreshConfig) cod.Handler {
	skiper := config.Skipper
	if skiper == nil {
		skiper = DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skiper(c) {
			return c.Next()
		}
		err = c.Next()
		if err != nil {
			return
		}
		// 如果空数据或者已经是304，则跳过
		if len(c.BodyBytes) == 0 || c.StatusCode == http.StatusNotModified {
			return
		}

		// 如果非GET HEAD请求，则跳过
		method := c.Request.Method
		if method != http.MethodGet && method != http.MethodHead {
			return
		}

		// 如果响应状态码非 >=200 <300，则跳过
		statusCode := c.StatusCode
		if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
			return
		}

		// 304的处理
		if fresh.Fresh(c.Request.Header, c.Headers) {
			c.NotModified()
		}
		return
	}
}
