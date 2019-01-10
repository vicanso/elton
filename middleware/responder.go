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

	jsoniter "github.com/json-iterator/go"
	"github.com/vicanso/cod"
	"github.com/vicanso/hes"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type (
	// ResponderConfig response config
	ResponderConfig struct {
		Skipper Skipper
	}
)

// NewResponder create a responder
func NewResponder(config ResponderConfig) cod.Handler {
	skiper := config.Skipper
	if skiper == nil {
		skiper = DefaultSkipper
	}
	return func(c *cod.Context) error {
		if skiper(c) {
			return c.Next()
		}
		e := c.Next()
		bodyBuf := c.BodyBuffer
		// 如果已生成BodyBytes，则无跳过
		// 无需要从 Body 中转换 BodyBytes
		if bodyBuf != nil {
			return e
		}
		var err *hes.Error
		if e != nil {
			// 如果出错，尝试转换为HTTPError
			he, ok := e.(*hes.Error)
			if !ok {
				he = &hes.Error{
					StatusCode: http.StatusInternalServerError,
					Message:    e.Error(),
				}
			}
			err = he
		}

		if err == nil && c.StatusCode == 0 && c.Body == nil {
			// 如果status code 与 body 都为空，则为非法响应
			err = cod.ErrInvalidResponse
		}

		ct := cod.HeaderContentType

		// 从出错中获取响应数据，响应状态码
		if err != nil {
			c.StatusCode = err.StatusCode
			c.Body, _ = json.Marshal(err)
			c.SetHeader(ct, cod.MIMEApplicationJSON)
		}

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
					body = []byte(c.Body.(string))
				}
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
