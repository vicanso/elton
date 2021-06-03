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
	"net/http"

	"github.com/vicanso/elton"
)

type (
	// FreshConfig fresh config
	FreshConfig struct {
		Skipper elton.Skipper
	}
)

// NewDefaultFresh returns a default fresh middleware, it will return 304 modified if the data is not modified.
func NewDefaultFresh() elton.Handler {
	return NewFresh(FreshConfig{})
}

// NewFresh returns a fresh middleware.
func NewFresh(config FreshConfig) elton.Handler {
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	return func(c *elton.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		err = c.Next()
		if err != nil {
			return
		}
		// 如果空数据或者已经是304，则跳过
		bodyBuf := c.BodyBuffer
		if bodyBuf == nil || bodyBuf.Len() == 0 || c.StatusCode == http.StatusNotModified {
			return
		}

		// 如果非GET HEAD请求，则跳过
		method := c.Request.Method
		if method != http.MethodGet && method != http.MethodHead {
			return
		}

		// 如果响应状态码不为0 而且( < 200 或者 >= 300)，则跳过
		// 如果未设置状态码，最终则为200
		statusCode := c.StatusCode
		if statusCode != 0 &&
			(statusCode < http.StatusOK ||
				statusCode >= http.StatusMultipleChoices) {
			return
		}

		// 304的处理
		if elton.Fresh(c.Request.Header, c.Header()) {
			c.NotModified()
		}
		return
	}
}
