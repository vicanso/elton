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
	"net/http"
	"strings"

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

type (
	// ErrorConfig error handler config
	ErrorConfig struct {
		Skipper      elton.Skipper
		ResponseType string
	}
)

const (
	// ErrErrorCategory error category of error handler
	ErrErrorCategory = "elton-error"
)

// NewDefaultError return a new error handler, it will convert the error to hes.Error and response.
// JSON will be used is client's request accept header support application/json, otherwise text will be used.
func NewDefaultError() elton.Handler {
	return NewError(ErrorConfig{})
}

// NewError return a new error handler.
func NewError(config ErrorConfig) elton.Handler {
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		err := c.Next()
		// 如果没有出错，直接返回
		if err == nil {
			return nil
		}
		he, ok := err.(*hes.Error)
		if !ok {
			he = hes.Wrap(err)
			// 非hes的error，则都认为是500出错异常
			he.StatusCode = http.StatusInternalServerError
			he.Exception = true
			he.Category = ErrErrorCategory
		}
		c.StatusCode = he.StatusCode
		if config.ResponseType == "json" ||
			strings.Contains(c.GetRequestHeader("Accept"), "application/json") {
			buf := he.ToJSON()
			c.BodyBuffer = bytes.NewBuffer(buf)
			c.SetHeader(elton.HeaderContentType, elton.MIMEApplicationJSON)
		} else {
			c.BodyBuffer = bytes.NewBufferString(he.Error())
			c.SetHeader(elton.HeaderContentType, elton.MIMETextPlain)
		}

		return nil
	}
}
