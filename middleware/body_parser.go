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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/vicanso/cod"
	"github.com/vicanso/hes"
)

const (
	// 默认为50kb
	defaultRequestBodyLimit   = 50 * 1024
	errBodyParserCategory     = "cod-body-parser"
	jsonContentType           = "application/json"
	formURLEncodedContentType = "application/x-www-form-urlencoded"
)

type (
	// BodyParserConfig json parser config
	BodyParserConfig struct {
		Limit                int
		IgnoreJSON           bool
		IgnoreFormURLEncoded bool
		Skipper              Skipper
	}
)

var (
	validMethods = []string{
		http.MethodPost,
		http.MethodPatch,
		http.MethodPut,
	}
)

// NewBodyParser new json parser
func NewBodyParser(config BodyParserConfig) cod.Handler {
	limit := defaultRequestBodyLimit
	if config.Limit != 0 {
		limit = config.Limit
	}
	skiper := config.Skipper
	if skiper == nil {
		skiper = DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skiper(c) || len(c.RequestBody) != 0 {
			return c.Next()
		}
		method := c.Request.Method

		// 对于非提交数据的method跳过
		valid := false
		for _, item := range validMethods {
			if item == method {
				valid = true
				break
			}
		}
		if !valid {
			return c.Next()
		}
		ct := c.Request.Header.Get(cod.HeaderContentType)
		// 非json则跳过
		isJSON := strings.HasPrefix(ct, jsonContentType)
		isFormURLEncoded := strings.HasPrefix(ct, formURLEncodedContentType)

		// 如果不是 json 也不是 form url encoded，则跳过
		if !isJSON && !isFormURLEncoded {
			return c.Next()
		}
		// 如果数据类型json，而且中间件不处理，则跳过
		if isJSON && config.IgnoreJSON {
			return c.Next()
		}

		// 如果数据类型form url encoded，而且中间件不处理，则跳过
		if isFormURLEncoded && config.IgnoreFormURLEncoded {
			return c.Next()
		}

		body, e := ioutil.ReadAll(c.Request.Body)
		if e != nil {
			err = &hes.Error{
				StatusCode: http.StatusBadRequest,
				Message:    e.Error(),
				Category:   errBodyParserCategory,
			}
			return
		}
		if limit > 0 && len(body) > limit {
			err = &hes.Error{
				StatusCode: http.StatusBadRequest,
				Message:    fmt.Sprintf("request body is %d bytes, it should be <= %d", len(body), limit),
				Category:   errBodyParserCategory,
			}
			return
		}
		// 将form url encoded 数据转化为json
		if isFormURLEncoded {
			values := strings.Split(string(body), "&")
			data := make([]string, len(values))
			for index, str := range values {
				arr := strings.Split(str, "=")
				v, _ := url.QueryUnescape(arr[1])
				if v == "" {
					v = arr[1]
				}
				data[index] = fmt.Sprintf(`"%s":"%s"`, arr[0], v)
			}
			body = []byte("{" + strings.Join(data, ",") + "}")
		}
		c.RequestBody = body
		return c.Next()
	}
}
