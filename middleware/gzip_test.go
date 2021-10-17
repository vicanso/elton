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
	"io/ioutil"
	"math"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestGzipCompress(t *testing.T) {
	assert := assert.New(t)
	originalData := randomString(1024)
	g := new(GzipCompressor)
	req := httptest.NewRequest("GET", "/users/me", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	c := elton.NewContext(nil, req)

	acceptable, encoding := g.Accept(c, 0)
	assert.False(acceptable)
	assert.Empty(encoding)

	acceptable, encoding = g.Accept(c, len(originalData))
	assert.True(acceptable)
	assert.Equal(GzipEncoding, encoding)

	_, err := g.Compress([]byte(originalData), 1)
	assert.Nil(err)
	buf, err := g.Compress([]byte(originalData), math.MinInt)
	assert.Nil(err)

	r, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
	assert.Nil(err)
	defer r.Close()
	originalBuf, _ := ioutil.ReadAll(r)
	assert.Equal(originalData, string(originalBuf))
}

func TestGzipPipe(t *testing.T) {
	assert := assert.New(t)
	resp := httptest.NewRecorder()
	originalData := randomString(1024)
	c := elton.NewContext(resp, nil)

	c.Body = bytes.NewReader([]byte(originalData))

	g := new(GzipCompressor)
	err := g.Pipe(c)
	assert.Nil(err)
	r, err := gzip.NewReader(resp.Body)
	assert.Nil(err)
	defer r.Close()
	buf, _ := ioutil.ReadAll(r)
	assert.Equal(originalData, string(buf))
}

func TestGzipGetLevel(t *testing.T) {
	assert := assert.New(t)
	g := GzipCompressor{
		Level: 1000,
	}
	assert.Equal(gzip.BestCompression, g.getLevel())

	g.Level = 0
	assert.Equal(gzip.DefaultCompression, g.getLevel())

	g.Level = 1
	assert.Equal(1, g.getLevel())
}

func TestGzipGetMinLength(t *testing.T) {
	assert := assert.New(t)
	g := GzipCompressor{}

	assert.Equal(DefaultCompressMinLength, g.getMinLength())

	g.MinLength = 1
	assert.Equal(1, g.getMinLength())
}
