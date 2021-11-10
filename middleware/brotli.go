// MIT License

// Copyright (c) 2021 Tree Xie

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
	"io"
	"io/ioutil"

	"github.com/andybalholm/brotli"
	"github.com/vicanso/elton"
)

const (
	// BrEncoding br encoding
	BrEncoding       = "br"
	maxBrQuality     = 11
	defaultBrQuality = 6
)

type (
	// BrCompressor brotli compress
	BrCompressor struct {
		Level     int
		MinLength int
	}
)

func (b *BrCompressor) getLevel() int {
	level := b.Level
	if level <= 0 {
		level = defaultBrQuality
	}
	if level > maxBrQuality {
		level = maxBrQuality
	}
	return level
}

func (b *BrCompressor) getMinLength() int {
	if b.MinLength == 0 {
		return DefaultCompressMinLength
	}
	return b.MinLength
}

// Accept check accept econding
func (b *BrCompressor) Accept(c *elton.Context, bodySize int) (acceptable bool, encoding string) {
	// 如果数据少于最低压缩长度，则不压缩
	if bodySize >= 0 && bodySize < b.getMinLength() {
		return
	}
	return AcceptEncoding(c, BrEncoding)
}

// BrotliCompress compress data by brotli
func BrotliCompress(buf []byte, level int) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	w := brotli.NewWriterLevel(buffer, level)
	_, err := w.Write(buf)
	if err != nil {
		return nil, err
	}
	// 直接调用close触发数据的flush
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

// BrotliDecompress decompress data of brotli
func BrotliDecompress(buf []byte) (*bytes.Buffer, error) {
	r := brotli.NewReader(bytes.NewBuffer(buf))
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(data), nil
}

// Compress brotli compress
func (b *BrCompressor) Compress(buf []byte, levels ...int) (*bytes.Buffer, error) {
	level := b.getLevel()
	if len(levels) != 0 && levels[0] != IgnoreCompression {
		level = levels[0]
	}
	return BrotliCompress(buf, level)
}

// Pipe brotli pipe
func (b *BrCompressor) Pipe(c *elton.Context) (err error) {
	r := c.Body.(io.Reader)
	closer, ok := c.Body.(io.Closer)
	if ok {
		defer closer.Close()
	}
	w := brotli.NewWriterLevel(c.Response, b.getLevel())

	defer w.Close()
	_, err = io.Copy(w, r)
	return
}
