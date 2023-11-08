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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBrotliCompress(t *testing.T) {
	assert := assert.New(t)
	compressor := NewCacheBrCompressor()
	compressor.MinLength = 20

	assert.False(compressor.IsValid("text", 1))
	assert.True(compressor.IsValid("text", 100))

	data := bytes.NewBufferString("hello world!hello world!hello world!")
	result, compressionType, err := compressor.Compress(data)
	assert.Nil(err)
	assert.Equal(CompressionBr, compressionType)
	assert.NotEqual(data, result)
	result, _ = BrotliDecompress(result.Bytes())
	assert.Equal(data, result)
}

func TestNewGzipCompress(t *testing.T) {
	assert := assert.New(t)
	compressor := NewCacheGzipCompressor()
	compressor.MinLength = 20

	assert.False(compressor.IsValid("text", 1))
	assert.True(compressor.IsValid("text", 100))

	data := bytes.NewBufferString("hello world!hello world!hello world!")
	result, compressionType, err := compressor.Compress(data)
	assert.Equal(CompressionGzip, compressionType)
	assert.Nil(err)
	assert.NotEqual(data, result)
	result, _ = GzipDecompress(result.Bytes())
	assert.Equal(data, result)
}

func TestNewZstdCompress(t *testing.T) {
	assert := assert.New(t)
	compressor := NewCacheZstdCompressor()
	compressor.MinLength = 20

	assert.False(compressor.IsValid("text", 1))
	assert.True(compressor.IsValid("text", 100))

	data := bytes.NewBufferString("hello world!hello world!hello world!")
	result, compressionType, err := compressor.Compress(data)
	assert.Equal(CompressionZstd, compressionType)
	assert.Nil(err)
	assert.NotEqual(data, result)
	result, _ = ZstdDecompress(result.Bytes())
	assert.Equal(data, result)
}
