// MIT License

// Copyright (c) 2021 Tree Xie

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
	"errors"
	"net/http"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/hes"
)

const (
	ErrResponseSizeLimiterCategory = "elton-response-size-limiter"
)

type (
	ResponseSizeLimiterConfig struct {
		Skipper elton.Skipper
		MaxSize int
	}
)

var ErrResponseTooLarge = &hes.Error{
	Category:   ErrResponseSizeLimiterCategory,
	Message:    "body of response is too large",
	StatusCode: http.StatusInternalServerError,
}

// NewResponseSizeLimiter returns a new response size limiter.
// 注意：仅对缓冲型响应（BodyBuffer）生效，流式reader body不做限制；
// 且校验发生在body已生成之后，用于阻止超大响应返回客户端，
// 而非阻止其占用服务端内存
func NewResponseSizeLimiter(config ResponseSizeLimiterConfig) elton.Handler {
	skipper := getSkipper(config.Skipper)
	if config.MaxSize <= 0 {
		panic(errors.New("max size should be > 0"))
	}
	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		err := c.Next()
		if err != nil {
			return err
		}
		if c.BodyBuffer != nil && c.BodyBuffer.Len() > config.MaxSize {
			return ErrResponseTooLarge
		}
		return nil
	}
}
