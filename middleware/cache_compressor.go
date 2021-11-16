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
)

type CompressionType uint8

const (
	// not compress
	CompressionNon CompressionType = iota
	// gzip compress
	CompressionGzip
	// br compress
	CompressionBr
)

type CacheDecompressor interface {
	Decompress(buffer *bytes.Buffer) (data *bytes.Buffer, err error)
	GetEncoding() (encoding string)
}

var cacheDecompressors = map[CompressionType]CacheDecompressor{}

// RegisterCacheDecompressor register cache decompressor
func RegisterCacheDecompressor(compressionType CompressionType, decompressor CacheDecompressor) {
	cacheDecompressors[compressionType] = decompressor
}

type brDecompressor struct{}

func (br *brDecompressor) Decompress(data *bytes.Buffer) (*bytes.Buffer, error) {
	return BrotliDecompress(data.Bytes())
}
func (br *brDecompressor) GetEncoding() string {
	return BrEncoding
}

type gzipDecompressor struct{}

func (g *gzipDecompressor) Decompress(data *bytes.Buffer) (*bytes.Buffer, error) {
	return GzipDecompress(data.Bytes())
}
func (g *gzipDecompressor) GetEncoding() string {
	return GzipEncoding
}

func init() {
	RegisterCacheDecompressor(CompressionBr, &brDecompressor{})
	RegisterCacheDecompressor(CompressionGzip, &gzipDecompressor{})
}

type CacheBodyCompressParams struct {
	Quality         int
	MinLength       int
	ContentTypeReg  *regexp.Regexp
	Compress        func(buf []byte, level int) (*bytes.Buffer, error)
	CompressionType CompressionType
}

// NewCacheBodyCompress creates a new compress for cache body
func NewCacheBodyCompress(params CacheBodyCompressParams) CacheBodyCompress {
	minLength := params.MinLength
	quality := params.Quality
	contentTypeReg := params.ContentTypeReg

	return func(buffer *bytes.Buffer, contentType string) (*bytes.Buffer, CompressionType, error) {
		// 如果buffer为空
		if buffer == nil ||
			// 数据长度少于最小压缩长度
			buffer.Len() < minLength ||
			// 未配置content type
			contentTypeReg == nil ||
			// 数据类型不匹配
			!contentTypeReg.MatchString(contentType) {
			return buffer, CompressionNon, nil
		}
		data, err := params.Compress(buffer.Bytes(), quality)
		if err != nil {
			return nil, CompressionNon, err
		}
		return data, params.CompressionType, nil
	}
}

// NewBrotliCompress creates a brotli compress function
func NewBrotliCompress(quality, minLength int, contentTypeReg *regexp.Regexp) CacheBodyCompress {
	return NewCacheBodyCompress(CacheBodyCompressParams{
		Quality:         quality,
		MinLength:       minLength,
		ContentTypeReg:  contentTypeReg,
		CompressionType: CompressionBr,
		Compress:        BrotliCompress,
	})
}

// NewGzipCompress creates a gzip compress function
func NewGzipCompress(quality, minLength int, contentTypeReg *regexp.Regexp) CacheBodyCompress {
	return NewCacheBodyCompress(CacheBodyCompressParams{
		Quality:         quality,
		MinLength:       minLength,
		ContentTypeReg:  contentTypeReg,
		CompressionType: CompressionGzip,
		Compress:        GzipCompress,
	})
}
