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
)

type (
	// ETagConfig eTag config
	ETagConfig struct {
		Skipper Skipper
	}
)

// NewETag create a eTag middleware
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
		// 如果无内容或已设置 eTag ，则跳过
		if len(c.BodyBytes) == 0 ||
			respHeader.Get(cod.HeaderETag) != "" {
			return
		}
		// 如果状态码非 >= 200 < 300 ，则跳过
		if c.StatusCode < http.StatusOK ||
			c.StatusCode >= http.StatusMultipleChoices {
			return
		}
		eTag := cod.GenerateETag(c.BodyBytes)
		c.SetHeader(cod.HeaderETag, eTag)
		return
	}
}
