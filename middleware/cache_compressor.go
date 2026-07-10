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
	"regexp"

	"github.com/vicanso/elton/v2"
)

type CompressionType uint8

// CompressionType 的取值会作为"压缩类型字节"持久化至缓存数据中，
// 取值必须保持稳定，新增类型只能追加，不可修改已有值的顺序
const (
	// not compress
	CompressionNone CompressionType = iota
	// gzip compress
	CompressionGzip
	// br compress
	CompressionBr
	// zstd compress
	CompressionZstd
)

type CacheCompressor interface {
	// decompress function
	Decompress(buffer *bytes.Buffer) (data *bytes.Buffer, err error)
	// get encoding of compressor
	Encoding() (encoding string)
	// is valid for compress
	IsValid(contentType string, length int) (valid bool)
	// compress function
	Compress(buffer *bytes.Buffer) (data *bytes.Buffer, compressionType CompressionType, err error)
	// get compression type
	Compression() CompressionType
}

func isValidForCompress(reg *regexp.Regexp, minLength int, contentType string, length int) bool {
	if length < minLength {
		return false
	}
	if reg == nil {
		reg = DefaultCompressRegexp
	}
	return reg.MatchString(contentType)
}

// CacheBrCompressor brotli compressor for cache,
// it embeds BrCompressor and shares its level and min length config
type CacheBrCompressor struct {
	BrCompressor
	ContentRegexp *regexp.Regexp
}

func NewCacheBrCompressor() *CacheBrCompressor {
	return &CacheBrCompressor{}
}

func (br *CacheBrCompressor) Decompress(data *bytes.Buffer) (*bytes.Buffer, error) {
	return BrotliDecompress(data.Bytes())
}
func (br *CacheBrCompressor) Encoding() string {
	return elton.Br
}
func (br *CacheBrCompressor) IsValid(contentType string, length int) bool {
	return isValidForCompress(br.ContentRegexp, br.getMinLength(), contentType, length)
}
func (br *CacheBrCompressor) Compress(buffer *bytes.Buffer) (*bytes.Buffer, CompressionType, error) {
	data, err := BrotliCompress(buffer.Bytes(), br.getLevel())
	if err != nil {
		return nil, CompressionNone, err
	}
	return data, br.Compression(), nil
}
func (br *CacheBrCompressor) Compression() CompressionType {
	return CompressionBr
}

// CacheGzipCompressor gzip compressor for cache,
// it embeds GzipCompressor and shares its level and min length config
type CacheGzipCompressor struct {
	GzipCompressor
	ContentRegexp *regexp.Regexp
}

func NewCacheGzipCompressor() *CacheGzipCompressor {
	return &CacheGzipCompressor{}
}

func (g *CacheGzipCompressor) Decompress(data *bytes.Buffer) (*bytes.Buffer, error) {
	return GzipDecompress(data.Bytes())
}
func (g *CacheGzipCompressor) Encoding() string {
	return elton.Gzip
}
func (g *CacheGzipCompressor) IsValid(contentType string, length int) bool {
	return isValidForCompress(g.ContentRegexp, g.getMinLength(), contentType, length)
}
func (g *CacheGzipCompressor) Compress(buffer *bytes.Buffer) (*bytes.Buffer, CompressionType, error) {
	data, err := GzipCompress(buffer.Bytes(), g.getLevel())
	if err != nil {
		return nil, CompressionNone, err
	}
	return data, g.Compression(), nil
}
func (g *CacheGzipCompressor) Compression() CompressionType {
	return CompressionGzip
}

// CacheZstdCompressor zstd compressor for cache,
// it embeds ZstdCompressor and shares its level and min length config
type CacheZstdCompressor struct {
	ZstdCompressor
	ContentRegexp *regexp.Regexp
}

func NewCacheZstdCompressor() *CacheZstdCompressor {
	return &CacheZstdCompressor{}
}
func (z *CacheZstdCompressor) Decompress(data *bytes.Buffer) (*bytes.Buffer, error) {
	return ZstdDecompress(data.Bytes())
}
func (z *CacheZstdCompressor) Encoding() string {
	return elton.Zstd
}
func (z *CacheZstdCompressor) IsValid(contentType string, length int) bool {
	return isValidForCompress(z.ContentRegexp, z.getMinLength(), contentType, length)
}
func (z *CacheZstdCompressor) Compress(buffer *bytes.Buffer) (*bytes.Buffer, CompressionType, error) {
	data, err := ZstdCompress(buffer.Bytes(), z.getLevel())
	if err != nil {
		return nil, CompressionNone, err
	}
	return data, z.Compression(), nil
}
func (z *CacheZstdCompressor) Compression() CompressionType {
	return CompressionZstd
}
