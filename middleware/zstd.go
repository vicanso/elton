// MIT License

// Copyright (c) 2023 Tree Xie

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

	"github.com/klauspost/compress/zstd"
	"github.com/vicanso/elton"
)

type (
	// ZstdCompressor zstd compress
	ZstdCompressor struct {
		Level     int
		MinLength int
	}
)

// Accept accept zstd encoding
func (z *ZstdCompressor) Accept(c *elton.Context, bodySize int) (bool, string) {

	// 如果数据少于最低压缩长度，则不压缩（因为reader中的bodySize会设置为1，因此需要判断>=0）
	if bodySize >= 0 && bodySize < z.getMinLength() {
		return false, ""
	}
	return AcceptEncoding(c, elton.Zstd)
}

func (z *ZstdCompressor) getLevel() int {
	level := z.Level
	if level <= 0 {
		level = int(zstd.SpeedBetterCompression)
	}
	if level > int(zstd.SpeedBestCompression) {
		level = int(zstd.SpeedBestCompression)
	}
	return level
}

func (z *ZstdCompressor) getMinLength() int {
	if z.MinLength == 0 {
		return DefaultCompressMinLength
	}
	return z.MinLength
}

// Compress compress data by zstd
func (z *ZstdCompressor) Compress(buf []byte, levels ...int) (*bytes.Buffer, error) {
	level := z.getLevel()
	if len(levels) != 0 && levels[0] != IgnoreCompression {
		level = levels[0]
	}
	return ZstdCompress(buf, level)
}

// Pipe compress by pipe
func (z *ZstdCompressor) Pipe(c *elton.Context) error {
	r := c.Body.(io.Reader)
	closer, ok := c.Body.(io.Closer)
	if ok {
		defer closer.Close()
	}
	enc, err := zstd.NewWriter(c.Response, zstd.WithEncoderLevel(zstd.EncoderLevel(z.getLevel())))
	if err != nil {
		return err
	}

	_, err = io.Copy(enc, r)
	if err != nil {
		enc.Close()
		return err
	}
	return enc.Close()
}

// ZstdCompressor compress data by zstd
func ZstdCompress(buf []byte, level int) (*bytes.Buffer, error) {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevel(level)))
	if err != nil {
		return nil, err
	}
	dst := encoder.EncodeAll(buf, make([]byte, 0, len(buf)))
	return bytes.NewBuffer(dst), nil
}

// ZstdDecompress decompress data of zstd
func ZstdDecompress(buf []byte) (*bytes.Buffer, error) {
	r, err := zstd.NewReader(bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(data), nil
}
