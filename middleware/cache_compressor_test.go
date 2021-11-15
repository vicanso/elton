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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBrotliCompress(t *testing.T) {
	assert := assert.New(t)
	fn := NewBrotliCompress(1, 20, regexp.MustCompile("text"))

	data := bytes.NewBufferString("hello world!")
	result, compressionType, err := fn(data, "text")
	assert.Nil(err)
	assert.Equal(data, result)
	assert.Equal(CompressionNon, compressionType)

	data = bytes.NewBufferString("hello world!hello world!hello world!")
	result, compressionType, err = fn(data, "text")
	assert.Nil(err)
	assert.NotEqual(data, result)
	assert.Equal(CompressionBr, compressionType)
	result, _ = BrotliDecompress(result.Bytes())
	assert.Equal(data, result)
}

func TestNewGzipCompress(t *testing.T) {
	assert := assert.New(t)
	fn := NewGzipCompress(1, 20, regexp.MustCompile("text"))

	data := bytes.NewBufferString("hello world!")
	result, compressionType, err := fn(data, "text")
	assert.Nil(err)
	assert.Equal(data, result)
	assert.Equal(CompressionNon, compressionType)

	data = bytes.NewBufferString("hello world!hello world!hello world!")
	result, compressionType, err = fn(data, "text")
	assert.Nil(err)
	assert.NotEqual(data, result)
	assert.Equal(CompressionGzip, compressionType)
	result, _ = GzipDecompress(result.Bytes())
	assert.Equal(data, result)
}
