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
	return AcceptEncoding(c, "test")
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

func TestCompressMiddleware(t *testing.T) {
	assert := assert.New(t)
	defaultCompress := NewDefaultCompress()
	next := func() error {
		return nil
	}
	randomData := randomString(4096)
	htmlData := "<html><body>" + randomString(8192) + "</body></html>"
	htmlGzip, _ := (&GzipCompressor{}).Compress([]byte(htmlData))

	customCompress := NewCompress(CompressConfig{
		Compressors: []Compressor{
			new(testCompressor),
		},
	})

	tests := []struct {
		newContext func() *elton.Context
		fn         elton.Handler
		err        error
		result     []byte
		encoding   string
		etag       string
	}{
		// committed true
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/", nil)
				c := elton.NewContext(httptest.NewRecorder(), req)
				c.Next = next
				c.Committed = true
				return c
			},
			fn: defaultCompress,
		},
		// no body
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/", nil)
				c := elton.NewContext(httptest.NewRecorder(), req)
				c.Next = next
				return c
			},
			fn: defaultCompress,
		},
		// error
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), nil)
				customErr := errors.New("abccd")
				c.Next = func() error {
					return customErr
				}
				return c
			},
			fn:  defaultCompress,
			err: errors.New("abccd"),
		},
		// already encoding
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/users/me", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				body := bytes.NewBufferString(randomData)
				c.BodyBuffer = body
				c.SetHeader(elton.HeaderContentEncoding, "custom encoding")
				c.Next = next
				return c
			},
			fn:       defaultCompress,
			result:   []byte(randomData),
			encoding: "custom encoding",
		},
		// data size is less the compress min length
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/users/me", nil)
				req.Header.Set(elton.HeaderAcceptEncoding, "gzip")
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				body := bytes.NewBufferString("abcd")
				c.BodyBuffer = body
				c.SetHeader(elton.HeaderContentType, "text/plain")
				c.Next = next
				return c
			},
			fn:     defaultCompress,
			result: []byte("abcd"),
		},
		// content type is not match
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/users/me", nil)
				req.Header.Set(elton.HeaderAcceptEncoding, "gzip")
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.SetHeader(elton.HeaderContentType, "image/jpeg")
				body := bytes.NewBufferString(randomData)
				c.BodyBuffer = body
				c.Next = next
				return c
			},
			fn:     defaultCompress,
			result: []byte(randomData),
		},
		// request not accept encoding
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/users/me", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.SetHeader(elton.HeaderContentType, "text/html")
				body := bytes.NewBufferString(randomData)
				c.BodyBuffer = body
				c.Next = next
				return c
			},
			fn:     defaultCompress,
			result: []byte(randomData),
		},
		// custom compress
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/users/me", nil)
				req.Header.Set("Accept-Encoding", "gzip, deflate, test")
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.SetHeader(elton.HeaderContentType, "text/html")
				c.BodyBuffer = bytes.NewBufferString("<html><body>" + randomString(8192) + "</body></html>")
				c.Next = next
				return c
			},
			fn:       customCompress,
			result:   []byte("abcd"),
			encoding: "test",
		},
		// update etag
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/users/me", nil)
				req.Header.Set("Accept-Encoding", "gzip")
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.SetHeader(elton.HeaderContentType, "text/html")
				c.SetHeader(elton.HeaderETag, "123")
				c.BodyBuffer = bytes.NewBufferString(htmlData)
				c.Next = next
				return c
			},
			fn:       defaultCompress,
			result:   htmlGzip.Bytes(),
			encoding: "gzip",
			etag:     "W/123",
		},
		// reader pike
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/users/me", nil)
				req.Header.Set(elton.HeaderAcceptEncoding, "gzip")
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.SetHeader(elton.HeaderContentType, "text/html")
				c.Next = func() error {
					return nil
				}
				body := bytes.NewBufferString(htmlData)
				c.Body = body
				c.Next = next
				return c
			},
			fn:       defaultCompress,
			encoding: "gzip",
		},
		// compress html
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/users/me", nil)
				req.Header.Set(elton.HeaderAcceptEncoding, "gzip")
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.SetHeader(elton.HeaderContentType, "text/html")
				c.Next = func() error {
					return nil
				}
				c.BodyBuffer = bytes.NewBufferString(htmlData)
				c.Next = next
				return c
			},
			fn:       defaultCompress,
			encoding: "gzip",
			result:   htmlGzip.Bytes(),
		},
	}
	for _, tt := range tests {
		c := tt.newContext()
		err := tt.fn(c)
		assert.Equal(tt.err, err)
		assert.Equal(tt.encoding, c.GetHeader(elton.HeaderContentEncoding))
		assert.Equal(tt.etag, c.GetHeader(elton.HeaderETag))
		if tt.result == nil {
			assert.Nil(c.BodyBuffer)
		} else {
			assert.Equal(tt.result, c.BodyBuffer.Bytes())
		}
	}
}
