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
	"strings"

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

const (
	// ErrRecoverCategory recover error category
	ErrRecoverCategory = "elton-recover"
)

// New new recover
func NewRecover() elton.Handler {
	return func(c *elton.Context) error {
		defer func() {
			// 可针对实际需求调整，如对于每个recover增加邮件通知等
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}

				he := hes.Wrap(err)
				he.Category = ErrRecoverCategory
				he.StatusCode = http.StatusInternalServerError
				err = he
				c.Elton().EmitError(c, err)
				// 出错时清除部分响应头
				for _, key := range []string{
					elton.HeaderETag,
					elton.HeaderLastModified,
					elton.HeaderContentEncoding,
					elton.HeaderContentLength,
				} {
					c.SetHeader(key, "")
				}
				// 直接对Response写入数据，则将 Committed设置为 true
				c.Committed = true
				resp := c.Response
				buf := []byte(err.Error())
				if strings.Contains(c.GetRequestHeader("Accept"), "application/json") {
					c.SetHeader(elton.HeaderContentType, elton.MIMEApplicationJSON)
					buf = he.ToJSON()
				}
				resp.WriteHeader(he.StatusCode)
				_, err = resp.Write(buf)
				if err != nil {
					c.Elton().EmitError(c, err)
				}
			}
		}()
		return c.Next()
	}
}
