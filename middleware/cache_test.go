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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestGetCacheMaxAge(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		key       string
		value     string
		age       int
		existsAge int
	}{
		// 设置了 set cookie
		{
			key:   elton.HeaderSetCookie,
			value: "set cookie",
			age:   0,
		},
		// 未设置cache control
		{
			age: 0,
		},
		// 设置了cache control 为 no cache
		{
			key:   elton.HeaderCacheControl,
			value: "no-cache",
			age:   0,
		},
		// 设置了cache control 为 no store
		{
			key:   elton.HeaderCacheControl,
			value: "no-store",
			age:   0,
		},
		// 设置了cache control 为 private
		{
			key:   elton.HeaderCacheControl,
			value: "private, max-age=10",
			age:   0,
		},
		// 设置了max-age
		{
			key:   elton.HeaderCacheControl,
			value: "max-age=10",
			age:   10,
		},
		// 设置了s-maxage
		{
			key:   elton.HeaderCacheControl,
			value: "max-age=10, s-maxage=1 ",
			age:   1,
		},
		// 设置了age
		{
			key:       elton.HeaderCacheControl,
			value:     "max-age=10",
			age:       8,
			existsAge: 2,
		},
	}

	for _, tt := range tests {
		h := http.Header{}
		h.Add(tt.key, tt.value)
		if tt.existsAge != 0 {
			h.Add("Age", strconv.Itoa(tt.existsAge))
		}
		age := GetCacheMaxAge(h)
		assert.Equal(tt.age, age)
	}
}

func TestCacheResponse(t *testing.T) {
	assert := assert.New(t)

	cp := &CacheResponse{
		Status:     StatusHitForPass,
		StatusCode: 200,
	}
	data := cp.Bytes()
	// hit for pass的只记录status
	assert.Equal(1, len(data))

	cp = &CacheResponse{
		Status:      StatusHit,
		CreatedAt:   1001,
		StatusCode:  200,
		Compression: CompressionGzip,
		Header: http.Header{
			"Cache-Control": []string{
				"no-cache",
			},
			"Content-Type": []string{
				"application/json",
			},
		},
		Body: bytes.NewBufferString("abcd"),
	}
	data = cp.Bytes()
	assert.Equal(43, len(data))

	cp = NewCacheResponse(data)
	assert.Equal(StatusHit, cp.Status)
	assert.Equal(uint32(1001), cp.CreatedAt)
	assert.Equal(200, cp.StatusCode)
	assert.Equal(CompressionGzip, cp.Compression)
	assert.Equal(http.Header{
		"Cache-Control": []string{
			"no-cache",
		},
		"Content-Type": []string{
			"application/json",
		},
	}, cp.Header)
	assert.Equal("abcd", cp.Body.String())
}

func TestCacheResponseGetBody(t *testing.T) {
	assert := assert.New(t)

	data := []byte("hello world!hello world!hello world!hello world!hello world!hello world!")
	brData, err := BrotliCompress(data, 1)
	assert.Nil(err)
	gzipData, err := GzipCompress(data, 1)
	assert.Nil(err)

	tests := []struct {
		newRespone     func() *CacheResponse
		acceptEncoding string
		encoding       string
		body           *bytes.Buffer
		compressor     CacheCompressor
	}{
		// 数据br, 客户端支持br
		{
			newRespone: func() *CacheResponse {
				return &CacheResponse{
					Compression: CompressionBr,
					Body:        brData,
				}
			},
			compressor:     NewCacheBrCompressor(),
			acceptEncoding: BrEncoding,
			encoding:       BrEncoding,
			body:           brData,
		},
		// 数据br, 客户端不支持br
		{
			newRespone: func() *CacheResponse {
				return &CacheResponse{
					Compression: CompressionBr,
					Body:        brData,
				}
			},
			compressor:     NewCacheBrCompressor(),
			acceptEncoding: "",
			encoding:       "",
			body:           bytes.NewBuffer(data),
		},
		// 数据gzip, 客户端支持gzip
		{
			newRespone: func() *CacheResponse {
				return &CacheResponse{
					Compression: CompressionGzip,
					Body:        gzipData,
				}
			},
			compressor:     NewCacheGzipCompressor(),
			acceptEncoding: GzipEncoding,
			encoding:       GzipEncoding,
			body:           gzipData,
		},
		// 数据gzip，客户端不支持gzip
		{
			newRespone: func() *CacheResponse {
				return &CacheResponse{
					Compression: CompressionGzip,
					Body:        gzipData,
				}
			},
			compressor:     NewCacheGzipCompressor(),
			acceptEncoding: "",
			encoding:       "",
			body:           bytes.NewBuffer(data),
		},
		// 数据非压缩
		{
			newRespone: func() *CacheResponse {
				return &CacheResponse{
					Compression: CompressionNone,
					Body:        bytes.NewBuffer(data),
				}
			},
			compressor:     NewCacheGzipCompressor(),
			acceptEncoding: "",
			encoding:       "",
			body:           bytes.NewBuffer(data),
		},
	}
	for _, tt := range tests {
		cp := tt.newRespone()
		body, encoding, err := cp.GetBody(tt.acceptEncoding, tt.compressor)
		assert.Nil(err)
		assert.Equal(tt.encoding, encoding)
		assert.Equal(tt.body, body)
	}
}

func TestCacheResponseSetBody(t *testing.T) {
	assert := assert.New(t)
	cp := CacheResponse{}
	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	c.SetRequestHeader(elton.HeaderAcceptEncoding, "br")

	// 无数据
	err := cp.SetBody(c, NewCacheBrCompressor())
	assert.Nil(err)
	assert.Nil(c.BodyBuffer)

	cp.Body = bytes.NewBufferString("hello world!")
	cp.Compression = CompressionBr
	err = cp.SetBody(c, NewCacheBrCompressor())
	assert.Nil(err)
	assert.Equal(bytes.NewBufferString("hello world!"), c.BodyBuffer)
	assert.Equal("br", c.GetHeader(elton.HeaderContentEncoding))
}

func TestIsPassCacheMethod(t *testing.T) {
	assert := assert.New(t)
	assert.False(IsPassCacheMethod("GET"))
	assert.False(IsPassCacheMethod("HEAD"))
	assert.True(IsPassCacheMethod("POST"))
}

func TestIsCacheable(t *testing.T) {
	assert := assert.New(t)
	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	c.SetHeader(elton.HeaderContentEncoding, "gzip")
	cacheable, cacheAge := isCacheable(c)
	assert.False(cacheable)
	assert.Equal(-2, cacheAge)
	c.Header().Del(elton.HeaderContentEncoding)

	cacheable, cacheAge = isCacheable(c)
	assert.False(cacheable)
	assert.Equal(-1, cacheAge)

	c.StatusCode = 200
	c.CacheMaxAge(time.Second)
	cacheable, cacheAge = isCacheable(c)
	assert.True(cacheable)
	assert.Equal(1, cacheAge)
}

type testStore struct {
	data sync.Map
}

func (ts *testStore) Get(ctx context.Context, key string) ([]byte, error) {
	value, ok := ts.data.Load(key)
	if !ok {
		return nil, nil
	}
	buf, _ := value.([]byte)
	return buf, nil
}

func (ts *testStore) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	ts.data.Store(key, data)
	return nil
}

func TestGetBodyBuffer(t *testing.T) {
	assert := assert.New(t)

	c := elton.NewContext(nil, nil)
	c.BodyBuffer = bytes.NewBufferString("abc")

	buffer, err := getBodyBuffer(c, nil)
	assert.Nil(err)
	assert.Equal(bytes.NewBufferString("abc"), buffer)

	c = elton.NewContext(nil, nil)
	c.Body = map[string]string{
		"name": "abc",
	}
	buffer, err = getBodyBuffer(c, json.Marshal)
	assert.Nil(err)
	assert.Equal(bytes.NewBufferString(`{"name":"abc"}`), buffer)
}

func TestNewCache(t *testing.T) {
	assert := assert.New(t)

	fn := NewDefaultCache(&testStore{})

	// POST 不可缓存
	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("POST", "/", nil))
	postErr := errors.New("post err")
	c.Next = func() error {
		return postErr
	}
	err := fn(c)
	assert.Equal(postErr, err)

	// fetch，结果hit for pass
	hitForPassReq := httptest.NewRequest("GET", "/?hitForPass", nil)
	c = elton.NewContext(httptest.NewRecorder(), hitForPassReq)
	c.Next = func() error {
		c.NoContent()
		return nil
	}
	err = fn(c)
	assert.Nil(err)
	assert.Equal(204, c.StatusCode)
	assert.Equal("fetch", c.GetHeader(HeaderXCache))

	// hit for pass
	c = elton.NewContext(httptest.NewRecorder(), hitForPassReq)
	hitForPassErr := errors.New("hit for pass")
	c.Next = func() error {
		return hitForPassErr
	}
	err = fn(c)
	assert.Equal(err, hitForPassErr)
	assert.Equal("hit-for-pass", c.GetHeader(HeaderXCache))

	// fetch，结果为可缓存
	cacheableReq := httptest.NewRequest("GET", "/?cacheable", nil)
	c = elton.NewContext(httptest.NewRecorder(), cacheableReq)
	c.SetRequestHeader(elton.HeaderAcceptEncoding, "br")
	buffer := &bytes.Buffer{}
	for i := 0; i < 1000; i++ {
		buffer.WriteString("hello world!")
	}
	c.Next = func() error {
		c.CacheMaxAge(time.Minute)
		c.SetContentTypeByExt(".txt")
		c.BodyBuffer = buffer
		return nil
	}
	err = fn(c)
	assert.Nil(err)
	assert.Equal("\x1b\xdf.\x00\xa4@Br\x90E\x1e\xcbe\xf2<\x9d\xda\xd1\x04 ", c.BodyBuffer.String())
	assert.Equal("fetch", c.GetHeader(HeaderXCache))
	decompressBuffer, err := BrotliDecompress(c.BodyBuffer.Bytes())
	assert.Nil(err)
	assert.Equal(buffer, decompressBuffer)

	// hit
	c = elton.NewContext(httptest.NewRecorder(), cacheableReq)
	// 设置了不缓存，在后续的缓存中间件会被清除
	c.NoCache()
	c.Next = func() error {
		// 直接读取缓存，不再调用next
		return errors.New("abc")
	}
	err = fn(c)
	assert.Nil(err)
	assert.Equal("\x1b\xdf.\x00\xa4@Br\x90E\x1e\xcbe\xf2<\x9d\xda\xd1\x04 ", c.BodyBuffer.String())
	assert.Equal("hit", c.GetHeader(HeaderXCache))
	assert.Equal([]string{
		"public, max-age=60",
	}, c.Header()["Cache-Control"])

	// hit（不支持压缩）
	c = elton.NewContext(httptest.NewRecorder(), cacheableReq)
	c.SetRequestHeader(elton.HeaderAcceptEncoding, "")
	c.Next = func() error {
		// 直接读取缓存，不再调用next
		return errors.New("abc")
	}
	err = fn(c)
	assert.Nil(err)
	assert.Equal(buffer, c.BodyBuffer)
	assert.Equal("hit", c.GetHeader(HeaderXCache))
}
