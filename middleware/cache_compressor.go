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
	"compress/gzip"
	"regexp"
)

type CompressionType uint8

const (
	// not compress
	CompressionNone CompressionType = iota
	// gzip compress
	CompressionGzip
	// br compress
	CompressionBr
)

type CacheCompressor interface {
	// decompress function
	Decompress(buffer *bytes.Buffer) (data *bytes.Buffer, err error)
	// get encoding of compressor
	GetEncoding() (encoding string)
	// is valid for compress
	IsValid(contentType string, length int) (valid bool)
	// compress function
	Compress(buffer *bytes.Buffer) (data *bytes.Buffer, compressionType CompressionType, err error)
	// get compression type
	GetCompression() CompressionType
}

type CacheBrCompressor struct {
	Level         int
	MinLength     int
	ContentRegexp *regexp.Regexp
}

func isValidForCompress(reg *regexp.Regexp, minLength int, contentType string, length int) bool {
	if minLength == 0 {
		minLength = DefaultCompressMinLength
	}
	if length < minLength {
		return false
	}
	if reg == nil {
		reg = DefaultCompressRegexp
	}
	return reg.MatchString(contentType)
}

func NewCacheBrCompressor() *CacheBrCompressor {
	return &CacheBrCompressor{
		Level: defaultBrQuality,
	}
}

func (br *CacheBrCompressor) Decompress(data *bytes.Buffer) (*bytes.Buffer, error) {
	return BrotliDecompress(data.Bytes())
}
func (br *CacheBrCompressor) GetEncoding() string {
	return BrEncoding
}
func (br *CacheBrCompressor) IsValid(contentType string, length int) bool {
	return isValidForCompress(br.ContentRegexp, br.MinLength, contentType, length)
}

func (br *CacheBrCompressor) Compress(buffer *bytes.Buffer) (*bytes.Buffer, CompressionType, error) {
	data, err := BrotliCompress(buffer.Bytes(), br.Level)
	if err != nil {
		return nil, CompressionNone, err
	}
	return data, CompressionBr, nil
}
func (br *CacheBrCompressor) GetCompression() CompressionType {
	return CompressionBr
}

type CacheGzipCompressor struct {
	Level         int
	MinLength     int
	ContentRegexp *regexp.Regexp
}

func NewCacheGzipCompressor() *CacheGzipCompressor {
	return &CacheGzipCompressor{
		Level: gzip.DefaultCompression,
	}
}

func (g *CacheGzipCompressor) Decompress(data *bytes.Buffer) (*bytes.Buffer, error) {
	return GzipDecompress(data.Bytes())
}
func (g *CacheGzipCompressor) GetEncoding() string {
	return GzipEncoding
}
func (g *CacheGzipCompressor) IsValid(contentType string, length int) bool {
	return isValidForCompress(g.ContentRegexp, g.MinLength, contentType, length)
}
func (g *CacheGzipCompressor) Compress(buffer *bytes.Buffer) (*bytes.Buffer, CompressionType, error) {
	data, err := GzipCompress(buffer.Bytes(), g.Level)
	if err != nil {
		return nil, CompressionNone, err
	}
	return data, CompressionGzip, nil
}
func (g *CacheGzipCompressor) GetCompression() CompressionType {
	return CompressionGzip
}
