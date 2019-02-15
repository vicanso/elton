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
	// Compressor compressor interface
	Compressor interface {
		Accept(c *cod.Context) (acceptable bool, encoding string)
		Compress([]byte, int) ([]byte, error)
	}
	// CompressConfig compress config
	CompressConfig struct {
		// Level 压缩率级别
		Level int
		// MinLength 最小压缩长度
		MinLength int
		// Checker 校验数据是否可压缩
		Checker        *regexp.Regexp
		Skipper        Skipper
		CompressorList []Compressor
	}
	// gzipCompressor gzip compress
	gzipCompressor struct{}
)

// AcceptEncoding check request accept encoding
func AcceptEncoding(c *cod.Context, encoding string) (bool, string) {
	acceptEncoding := c.GetRequestHeader(cod.HeaderAcceptEncoding)
	if strings.Contains(acceptEncoding, encoding) {
		return true, encoding
	}
	return false, ""
}

func (g *gzipCompressor) Accept(c *cod.Context) (acceptable bool, encoding string) {
	return AcceptEncoding(c, "gzip")
}

func (g *gzipCompressor) Compress(buf []byte, level int) ([]byte, error) {
	return doGzip(buf, level)
}

func addGzip(items []Compressor) []Compressor {
	return append(items, new(gzipCompressor))
}

// NewDefaultCompress create a default compress middleware, support gzip
func NewDefaultCompress() cod.Handler {
	return NewCompress(CompressConfig{})
}

// NewCompress create a new compress middleware
func NewCompress(config CompressConfig) cod.Handler {
	minLength := config.MinLength
	if minLength == 0 {
		minLength = defaultCompresMinLength
	}
	skipper := config.Skipper
	if skipper == nil {
		skipper = DefaultSkipper
	}
	checker := config.Checker
	if checker == nil {
		checker = defaultCompressRegexp
	}
	compressorList := config.CompressorList
	if compressorList == nil {
		compressorList = make([]Compressor, 0)
	}
	// 添加默认的 gzip 压缩
	compressorList = addGzip(compressorList)
	return func(c *cod.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		err = c.Next()
		if err != nil {
			return
		}

		bodyBuf := c.BodyBuffer
		// 如果数据为空，直接跳过
		if bodyBuf == nil {
			return
		}

		respHeader := c.Headers
		// encoding 不为空，已做处理，无需要压缩
		if respHeader.Get(cod.HeaderContentEncoding) != "" {
			return
		}
		contentType := respHeader.Get(cod.HeaderContentType)
		buf := bodyBuf.Bytes()
		// 如果数据长度少于最小压缩长度或数据类型为非可压缩，则返回
		if len(buf) < minLength || !checker.MatchString(contentType) {
			return
		}

		done := false
		for _, compressor := range compressorList {
			if done {
				break
			}
			acceptable, encoding := compressor.Accept(c)
			if !acceptable {
				continue
			}
			newBuf, e := compressor.Compress(buf, config.Level)
			// 如果压缩成功，则使用压缩数据
			// 失败则忽略
			if e == nil {
				c.SetHeader(cod.HeaderContentEncoding, encoding)
				bodyBuf.Reset()
				bodyBuf.Write(newBuf)
				done = true
			}
		}
		return
	}
}
