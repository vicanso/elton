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
	"compress/gzip"
	"io"

	"github.com/vicanso/elton"
)

const (
	// GzipEncoding gzip encoding
	GzipEncoding = "gzip"
)

type (
	// GzipCompressor gzip compress
	GzipCompressor struct {
		Level     int
		MinLength int
	}
)

// Accept accept gzip encoding
func (g *GzipCompressor) Accept(c *elton.Context, bodySize int) (bool, string) {
	// 如果数据少于最低压缩长度，则不压缩（因为reader中的bodySize会设置为1，因此需要判断>=0）
	if bodySize >= 0 && bodySize < g.getMinLength() {
		return false, ""
	}
	return AcceptEncoding(c, GzipEncoding)
}

// Compress compress data by gzip
func (g *GzipCompressor) Compress(buf []byte, levels ...int) (*bytes.Buffer, error) {
	level := g.getLevel()
	if len(levels) != 0 && levels[0] != IgnoreCompression {
		level = levels[0]
	}
	buffer := new(bytes.Buffer)

	w, err := gzip.NewWriterLevel(buffer, level)
	if err != nil {
		return nil, err
	}
	_, err = w.Write(buf)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func (g *GzipCompressor) getLevel() int {
	level := g.Level
	if level <= 0 {
		level = gzip.DefaultCompression
	}
	if level > gzip.BestCompression {
		level = gzip.BestCompression
	}
	return level
}

func (g *GzipCompressor) getMinLength() int {
	if g.MinLength == 0 {
		return DefaultCompressMinLength
	}
	return g.MinLength
}

// Pipe compress by pipe
func (g *GzipCompressor) Pipe(c *elton.Context) error {
	r := c.Body.(io.Reader)
	closer, ok := c.Body.(io.Closer)
	if ok {
		defer closer.Close()
	}
	w, _ := gzip.NewWriterLevel(c.Response, g.getLevel())
	defer w.Close()
	_, err := io.Copy(w, r)
	return err
}
