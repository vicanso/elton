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
	"regexp"
	"strings"

	"github.com/vicanso/cod"
)

var (
	defaultCompressRegexp = regexp.MustCompile("text|javascript|json")
)

const (
	defaultCompresMinLength = 1024
)

type (
	// Compress compress function
	Compress func([]byte, int) ([]byte, error)
	// Compression compression
	Compression struct {
		Type     string
		Compress Compress
	}
	// CompressConfig compress config
	CompressConfig struct {
		// Level 压缩率级别
		Level int
		// MinLength 最小压缩长度
		MinLength int
		// Checker 校验数据是否可压缩
		Checker         *regexp.Regexp
		Skipper         Skipper
		CompressionList []*Compression
	}
)

func addGzip(items []*Compression) []*Compression {
	found := false
	for _, c := range items {
		if c.Type == cod.Gzip {
			found = true
		}
	}
	if !found {
		items = append(items, &Compression{
			Type:     cod.Gzip,
			Compress: doGzip,
		})
	}
	return items

}

// NewCompresss create a new compress middleware
func NewCompresss(config CompressConfig) cod.Handler {
	minLength := config.MinLength
	if minLength == 0 {
		minLength = defaultCompresMinLength
	}
	skiper := config.Skipper
	if skiper == nil {
		skiper = DefaultSkipper
	}
	checker := config.Checker
	if checker == nil {
		checker = defaultCompressRegexp
	}
	compressionList := config.CompressionList
	if compressionList == nil {
		compressionList = make([]*Compression, 0)
	}
	// 添加默认的 gzip 压缩
	compressionList = addGzip(compressionList)
	return func(c *cod.Context) (err error) {
		if skiper(c) {
			return c.Next()
		}
		err = c.Next()
		if err != nil {
			return
		}
		respHeader := c.Headers
		encoding := respHeader.Get(cod.HeaderContentEncoding)
		// encoding 不为空，已做处理，无需要压缩
		if encoding != "" {
			return
		}
		contentType := respHeader.Get(cod.HeaderContentType)
		bodyBuf := c.BodyBuffer
		// 如果数据为空，直接跳过
		if bodyBuf == nil {
			return
		}
		buf := bodyBuf.Bytes()
		// 如果数据长度少于最小压缩长度或数据类型为非可压缩，则返回
		if len(buf) < minLength || !checker.MatchString(contentType) {
			return
		}

		acceptEncoding := c.GetRequestHeader(cod.HeaderAcceptEncoding)

		done := false
		for _, compression := range compressionList {
			if done {
				break
			}
			t := compression.Type
			// 如果请求端不接受当前压缩的类型
			if !strings.Contains(acceptEncoding, t) {
				continue
			}
			newBuf, e := compression.Compress(buf, config.Level)
			// 如果压缩成功，则使用压缩数据
			// 失败则忽略
			if e == nil {
				c.SetHeader(cod.HeaderContentEncoding, t)
				bodyBuf.Reset()
				bodyBuf.Write(newBuf)
				done = true
			}
		}

		return
	}
}
