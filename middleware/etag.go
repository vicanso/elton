package middleware

import (
	"net/http"

	"github.com/vicanso/cod"
)

type (
	// ETagConfig etag config
	ETagConfig struct {
		Skipper Skipper
	}
)

// NewETag create a etag middleware
func NewETag(config ETagConfig) cod.Handler {
	skiper := config.Skipper
	if skiper == nil {
		skiper = DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skiper(c) {
			return c.Next()
		}
		err = c.Next()
		respHeader := c.Headers
		// 如果无内容或已设置etag，则跳过
		if len(c.BodyBytes) == 0 ||
			respHeader.Get(cod.HeaderETag) != "" {
			return
		}
		// 如果状态码非 >= 200 < 300 ，则跳过
		if c.StatusCode < http.StatusOK ||
			c.StatusCode >= http.StatusMultipleChoices {
			return
		}
		etag := cod.GenerateETag(c.BodyBytes)
		c.SetHeader(cod.HeaderETag, etag)
		return
	}
}
