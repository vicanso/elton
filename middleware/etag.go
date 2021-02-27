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
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/vicanso/elton"
)

type (
	// ETagConfig ETag config
	ETagConfig struct {
		Skipper elton.Skipper
	}
)

// gen generate eTag
func genETag(buf []byte) (string, error) {
	size := len(buf)
	if size == 0 {
		return `"0-2jmj7l5rSw0yVb_vlWAYkK_YBwk="`, nil
	}
	h := sha1.New()
	_, err := h.Write(buf)
	if err != nil {
		return "", err
	}
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf(`"%x-%s"`, size, hash), nil
}

// NewDefaultETag returns a default ETag middleware, it will use sha1 to generate etag.
func NewDefaultETag() elton.Handler {
	return NewETag(ETagConfig{})
}

// NewETag returns a default ETag middleware.
func NewETag(config ETagConfig) elton.Handler {
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	return func(c *elton.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		err = c.Next()
		if err != nil {
			return
		}
		bodyBuf := c.BodyBuffer
		// 如果无内容或已设置 ETag ，则跳过
		// 因为没有内容也不生成 ETag
		if bodyBuf == nil || bodyBuf.Len() == 0 ||
			c.GetHeader(elton.HeaderETag) != "" {
			return
		}
		// 如果响应状态码不为0 而且( < 200 或者 >= 300)，则跳过
		// 如果未设置状态码，最终为200
		statusCode := c.StatusCode
		if statusCode != 0 &&
			(statusCode < http.StatusOK ||
				statusCode >= http.StatusMultipleChoices) {
			return
		}
		eTag, _ := genETag(bodyBuf.Bytes())
		if eTag != "" {
			c.SetHeader(elton.HeaderETag, eTag)
		}
		return
	}
}
