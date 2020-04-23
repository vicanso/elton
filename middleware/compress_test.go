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
	"errors"
	"math/rand"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

var letterRunes = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")

type testCompressor struct{}

func (t *testCompressor) Accept(c *elton.Context, bodySize int) (acceptable bool, encoding string) {
	return AcceptEncoding(c, "br")
}

func (t *testCompressor) Compress(buf []byte) (*bytes.Buffer, error) {
	return bytes.NewBufferString("abcd"), nil
}

func (t *testCompressor) Pipe(c *elton.Context) error {
	return nil
}

// randomString get random string
func randomString(n int) string {
	b := make([]rune, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func TestAcceptEncoding(t *testing.T) {
	assert := assert.New(t)
	req := httptest.NewRequest("GET", "/", nil)
	c := elton.NewContext(nil, req)
	acceptable, encoding := AcceptEncoding(c, elton.Gzip)
	assert.False(acceptable)
	assert.Empty(encoding)

	c.SetRequestHeader(elton.HeaderAcceptEncoding, elton.Gzip)
	acceptable, encoding = AcceptEncoding(c, elton.Gzip)
	assert.True(acceptable)
	assert.Equal(elton.Gzip, encoding)
}

func TestNewCompressConfig(t *testing.T) {
	assert := assert.New(t)
	conf := NewCompressConfig()
	assert.Empty(conf.Compressors)

	gzipCompressor := new(GzipCompressor)
	conf = NewCompressConfig(gzipCompressor)
	assert.Equal(1, len(conf.Compressors))
	assert.Equal(gzipCompressor, conf.Compressors[0])
}

func TestCompress(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		assert := assert.New(t)
		c := elton.NewContext(nil, nil)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		fn := NewCompress(CompressConfig{
			Skipper: func(c *elton.Context) bool {
				return true
			},
		})
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
	})

	t.Run("nil body", func(t *testing.T) {
		assert := assert.New(t)
		c := elton.NewContext(nil, nil)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		fn := NewDefaultCompress()
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
	})

	t.Run("return error", func(t *testing.T) {
		assert := assert.New(t)
		c := elton.NewContext(nil, nil)
		customErr := errors.New("abccd")
		c.Next = func() error {
			return customErr
		}
		fn := NewDefaultCompress()
		err := fn(c)
		assert.Equal(customErr, err)
	})

	t.Run("normal", func(t *testing.T) {
		assert := assert.New(t)
		conf := NewCompressConfig(&GzipCompressor{
			MinLength: 1,
		})
		fn := NewCompress(conf)

		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(elton.HeaderAcceptEncoding, "gzip")
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.SetHeader(elton.HeaderContentType, "text/html")
		c.BodyBuffer = bytes.NewBuffer([]byte("<html><body>" + randomString(8192) + "</body></html>"))
		originalSize := c.BodyBuffer.Len()
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
		assert.True(c.BodyBuffer.Len() < originalSize)
		assert.Equal(elton.Gzip, c.GetHeader(elton.HeaderContentEncoding))
	})

	t.Run("encoding done", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewDefaultCompress()
		req := httptest.NewRequest("GET", "/users/me", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		body := bytes.NewBufferString(randomString(4096))
		c.BodyBuffer = body
		c.SetHeader(elton.HeaderContentEncoding, "gzip")
		err := fn(c)
		assert.Nil(err)
		assert.Equal(body.Bytes(), c.BodyBuffer.Bytes())
	})

	t.Run("body size is less than min length", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewDefaultCompress()

		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(elton.HeaderAcceptEncoding, "gzip")
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		body := bytes.NewBufferString("abcd")
		c.BodyBuffer = body
		c.SetHeader(elton.HeaderContentType, "text/plain")
		err := fn(c)
		assert.Nil(err)
		assert.Equal(body.Bytes(), c.BodyBuffer.Bytes())
		assert.Empty(c.GetHeader(elton.HeaderContentEncoding))
	})

	t.Run("image should not be compress", func(t *testing.T) {
		assert := assert.New(t)

		fn := NewDefaultCompress()

		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(elton.HeaderAcceptEncoding, "gzip")
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.SetHeader(elton.HeaderContentType, "image/jpeg")
		c.Next = func() error {
			return nil
		}
		body := bytes.NewBufferString(randomString(4096))
		c.BodyBuffer = body
		err := fn(c)
		assert.Nil(err)
		assert.Equal(body.Bytes(), c.BodyBuffer.Bytes())
		assert.Empty(c.GetHeader(elton.HeaderContentEncoding))
	})

	t.Run("not accept gzip should not compress", func(t *testing.T) {
		assert := assert.New(t)

		fn := NewDefaultCompress()

		req := httptest.NewRequest("GET", "/users/me", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.SetHeader(elton.HeaderContentType, "text/html")
		c.Next = func() error {
			return nil
		}
		body := bytes.NewBufferString(randomString(4096))
		c.BodyBuffer = body
		err := fn(c)
		assert.Nil(err)
		assert.Equal(body.Bytes(), c.BodyBuffer.Bytes())
		assert.Empty(c.GetHeader(elton.HeaderContentEncoding))
	})

	t.Run("custom compress", func(t *testing.T) {
		assert := assert.New(t)
		compressorList := make([]Compressor, 0)
		compressorList = append(compressorList, new(testCompressor))
		fn := NewCompress(CompressConfig{
			Compressors: compressorList,
		})

		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.SetHeader(elton.HeaderContentType, "text/html")
		c.BodyBuffer = bytes.NewBufferString("<html><body>" + randomString(8192) + "</body></html>")
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
		assert.Equal(4, c.BodyBuffer.Len())
		assert.Equal("br", c.GetHeader(elton.HeaderContentEncoding))
	})

	t.Run("update etag", func(t *testing.T) {
		assert := assert.New(t)
		compressorList := make([]Compressor, 0)
		compressorList = append(compressorList, new(GzipCompressor))
		fn := NewCompress(CompressConfig{
			Compressors: compressorList,
		})

		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.SetHeader(elton.HeaderContentType, "text/html")
		c.SetHeader(elton.HeaderETag, "123")
		c.BodyBuffer = bytes.NewBufferString("<html><body>" + randomString(8192) + "</body></html>")
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
		assert.Equal("W/123", c.GetHeader(elton.HeaderETag))
	})

	t.Run("reader body", func(t *testing.T) {
		assert := assert.New(t)

		fn := NewDefaultCompress()

		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(elton.HeaderAcceptEncoding, "gzip")
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.SetHeader(elton.HeaderContentType, "text/html")
		c.Next = func() error {
			return nil
		}
		body := bytes.NewBufferString(randomString(4096))
		c.SetHeader(elton.HeaderContentLength, "4096")
		c.Body = body
		err := fn(c)
		assert.True(c.Committed)
		assert.Nil(err)
		assert.NotEmpty(resp.Body.Bytes())
		assert.Equal(elton.Gzip, c.GetHeader(elton.HeaderContentEncoding))
	})
}
