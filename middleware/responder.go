// Copyright 2018 tree xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package middleware

import (
	"bytes"
	"net/http"

	"github.com/vicanso/cod"
	"github.com/vicanso/hes"
)

type (
	// ResponderConfig response config
	ResponderConfig struct {
		Skipper Skipper
	}
)

var (
	// errInvalidResponse invalid response(body an status is nil)
	errInvalidResponse = &hes.Error{
		StatusCode: 500,
		Message:    "invalid response",
		Category:   ErrCategoryResponder,
	}
)

// NewDefaultResponder create a default responder
func NewDefaultResponder() cod.Handler {
	return NewResponder(ResponderConfig{})
}

// NewResponder create a responder
func NewResponder(config ResponderConfig) cod.Handler {
	skipper := config.Skipper
	if skipper == nil {
		skipper = DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		err = c.Next()
		if err != nil {
			return
		}
		// 如果已设置了BodyBuffer，则已生成好响应数据，跳过
		if c.BodyBuffer != nil {
			return
		}

		if c.StatusCode == 0 && c.Body == nil {
			// 如果status code 与 body 都为空，则为非法响应
			err = errInvalidResponse
			return
		}

		ct := cod.HeaderContentType

		hadContentType := false
		// 判断是否已设置响应头的Content-Type
		if c.GetHeader(ct) != "" {
			hadContentType = true
		}

		if c.StatusCode == 0 {
			c.StatusCode = http.StatusOK
		}
		statusCode := c.StatusCode

		var body []byte
		if c.Body != nil {
			switch c.Body.(type) {
			case string:
				if !hadContentType {
					c.SetHeader(ct, cod.MIMETextPlain)
				}
				body = []byte(c.Body.(string))
			case []byte:
				if !hadContentType {
					c.SetHeader(ct, cod.MIMEBinary)
				}
				body = c.Body.([]byte)
			default:
				// 转换为json
				buf, err := json.Marshal(c.Body)
				if err != nil {
					statusCode = http.StatusInternalServerError
					body = []byte(err.Error())
				} else {
					if !hadContentType {
						c.SetHeader(ct, cod.MIMEApplicationJSON)
					}
					body = buf
				}
			}
		}
		c.BodyBuffer = bytes.NewBuffer(body)
		c.StatusCode = statusCode

		return nil
	}
}
