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
