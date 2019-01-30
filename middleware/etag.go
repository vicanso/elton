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

// NewDefaultETag create a default ETag middleware
func NewDefaultETag() cod.Handler {
	return NewETag(ETagConfig{})
}

// NewETag create a ETag middleware
func NewETag(config ETagConfig) cod.Handler {
	skipper := config.Skipper
	if skipper == nil {
		skipper = DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		err = c.Next()
		respHeader := c.Headers
		bodyBuf := c.BodyBuffer
		// 如果无内容或已设置 ETag ，则跳过
		// 因为没有内容也不生成 ETag
		if bodyBuf == nil || bodyBuf.Len() == 0 ||
			respHeader.Get(cod.HeaderETag) != "" {
			return
		}
		// 如果状态码< 200 或者 >= 300 ，则跳过
		if c.StatusCode < http.StatusOK ||
			c.StatusCode >= http.StatusMultipleChoices {
			return
		}
		eTag := cod.GenerateETag(bodyBuf.Bytes())
		c.SetHeader(cod.HeaderETag, eTag)
		return
	}
}
