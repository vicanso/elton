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
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

type (
	// Config responder config
	ResponderConfig struct {
		Skipper elton.Skipper
		// Marshal custom marshal function
		Marshal func(v interface{}) ([]byte, error)
		// ContentType response's content type
		ContentType string
	}
)

const (
	// ErrResponderCategory responder error category
	ErrResponderCategory = "elton-responder"
)

var (
	// ErrInvalidResponse invalid response(body an status is nil)
	ErrInvalidResponse = &hes.Error{
		Exception:  true,
		StatusCode: 500,
		Message:    "invalid response",
		Category:   ErrResponderCategory,
	}
)

// NewDefaultResponder returns a new default responder middleware, it will use json.Marshal and application/json for response.
func NewDefaultResponder() elton.Handler {
	return NewResponder(ResponderConfig{})
}

// NewResponder returns a new responder middleware.
// If will use json.Marshal as default marshal function.
// If will use application/json as default content type.
func NewResponder(config ResponderConfig) elton.Handler {
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	marshal := config.Marshal
	// 如果未定义marshal
	if marshal == nil {
		marshal = json.Marshal
	}
	contentType := config.ContentType
	if contentType == "" {
		contentType = elton.MIMEApplicationJSON
	}

	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		err := c.Next()
		if err != nil {
			return err
		}
		// 如果已设置了BodyBuffer，则已生成好响应数据，跳过
		// 如果设置为commit，则表示其响应数据已处理
		if c.BodyBuffer != nil || c.Committed {
			return nil
		}

		if c.StatusCode == 0 && c.Body == nil {
			// 如果status code 与 body 都为空，则为非法响应
			return ErrInvalidResponse
		}
		// 如果body是reader，则跳过
		if c.IsReaderBody() {
			return nil
		}

		// 判断是否已设置响应头的Content-Type
		hadContentType := c.GetHeader(elton.HeaderContentType) != ""

		var body []byte
		if c.Body != nil {
			switch data := c.Body.(type) {
			case string:
				if !hadContentType {
					c.SetHeader(elton.HeaderContentType, elton.MIMETextPlain)
				}
				body = []byte(data)
			case []byte:
				if !hadContentType {
					c.SetHeader(elton.HeaderContentType, elton.MIMEBinary)
				}
				body = data
			default:
				// 使用marshal转换（默认为转换为json）
				buf, e := marshal(data)
				if e != nil {
					he := hes.NewWithErrorStatusCode(e, http.StatusInternalServerError)
					he.Category = ErrResponderCategory
					he.Exception = true
					return he
				}
				if !hadContentType {
					c.SetHeader(elton.HeaderContentType, contentType)
				}
				body = buf
			}
		}

		statusCode := c.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		if len(body) != 0 {
			c.BodyBuffer = bytes.NewBuffer(body)
		}
		c.StatusCode = statusCode
		return nil
	}
}
