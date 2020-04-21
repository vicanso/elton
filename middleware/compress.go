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
	"bytes"
	"regexp"
	"strings"

	"github.com/vicanso/elton"
)

var (
	DefaultCompressRegexp = regexp.MustCompile("text|javascript|json")
)

const (
	DefaultCompressMinLength = 1024
)

type (
	// Compressor compressor interface
	Compressor interface {
		// Accept accept check function
		Accept(c *elton.Context, bodySize int) (acceptable bool, encoding string)
		// Compress compress function
		Compress([]byte) (*bytes.Buffer, error)
		// Pipe pipe function
		Pipe(*elton.Context) error
	}
	// Config compress config
	CompressConfig struct {
		// Checker check the data is compressable
		Checker *regexp.Regexp
		// Compressors compressor list
		Compressors []Compressor
		// Skipper skipper function
		Skipper elton.Skipper
	}
)

// AcceptEncoding check request accept encoding
func AcceptEncoding(c *elton.Context, encoding string) (bool, string) {
	acceptEncoding := c.GetRequestHeader(elton.HeaderAcceptEncoding)
	if strings.Contains(acceptEncoding, encoding) {
		return true, encoding
	}
	return false, ""
}

// AddCompressor add compressor
func (conf *CompressConfig) AddCompressor(compressor Compressor) {
	if conf.Compressors == nil {
		conf.Compressors = make([]Compressor, 0)
	}
	conf.Compressors = append(conf.Compressors, compressor)
}

// NewCompressConfig create a new compress config
func NewCompressConfig(compressors ...Compressor) CompressConfig {
	cfg := CompressConfig{}
	cfg.AddCompressor(new(GzipCompressor))
	return cfg
}

// NewDefaultCompress create a new default compress middleware(include gzip)
func NewDefaultCompress() elton.Handler {
	cfg := NewCompressConfig(new(GzipCompressor))
	return NewCompress(cfg)
}

// NewCompress create a new compress middleware
func NewCompress(config CompressConfig) elton.Handler {
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	checker := config.Checker
	if checker == nil {
		checker = DefaultCompressRegexp
	}
	compressorList := config.Compressors
	return func(c *elton.Context) (err error) {
		if skipper(c) || compressorList == nil {
			return c.Next()
		}
		err = c.Next()
		if err != nil {
			return
		}
		isReaderBody := c.IsReaderBody()
		// 如果数据为空，而且body不是reader，直接跳过
		if c.BodyBuffer == nil && !isReaderBody {
			return
		}

		// encoding 不为空，已做处理，无需要压缩
		if c.GetHeader(elton.HeaderContentEncoding) != "" {
			return
		}
		contentType := c.GetHeader(elton.HeaderContentType)
		// 数据类型为非可压缩，则返回
		if !checker.MatchString(contentType) {
			return
		}

		var body []byte
		if c.BodyBuffer != nil {
			body = c.BodyBuffer.Bytes()
		}
		// 对于reader类，无法判断长度，认为长度为-1
		bodySize := -1
		if !isReaderBody {
			// 如果数据长度少于最小压缩长度
			bodySize = len(body)
		}

		fillHeader := func(encoding string) {
			c.SetHeader(elton.HeaderContentEncoding, encoding)
			c.AddHeader("Vary", "Accept-Encoding")
			etagValue := c.GetHeader(elton.HeaderETag)
			// after compress, etag should be weak etag
			if etagValue != "" && !strings.HasPrefix(etagValue, "W/") {
				c.SetHeader(elton.HeaderETag, "W/"+etagValue)
			}
		}

		for _, compressor := range compressorList {
			acceptable, encoding := compressor.Accept(c, bodySize)
			if !acceptable {
				continue
			}
			if isReaderBody {
				fillHeader(encoding)
				err = compressor.Pipe(c)
				// 如果出错直接返回
				if err != nil {
					return
				}
				// 成功跳出循环
				// pipe 将数据直接转至原有的Response，因此设置committed为true
				c.Committed = true
				// 清除 reader body
				c.Body = nil
				break
			}

			newBuf, e := compressor.Compress(body)
			// 如果压缩成功，则使用压缩数据
			// 失败则忽略
			if e == nil {
				fillHeader(encoding)
				c.BodyBuffer = newBuf
				break
			}
		}
		return
	}
}