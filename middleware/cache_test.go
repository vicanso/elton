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
		StatusCode: 200,
	}
	data := cp.Bytes()
	assert.Equal(11, len(data))

	cp = &CacheResponse{
		Status:     StatusHitForPass,
		CreatedAt:  1001,
		StatusCode: 200,
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
	assert.Equal(42, len(data))

	cp = NewCacheResponse(data)
	assert.Equal(StatusHitForPass, cp.Status)
	assert.Equal(uint32(1001), cp.CreatedAt)
	assert.Equal(200, cp.StatusCode)
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

func TestNewCache(t *testing.T) {
	assert := assert.New(t)

	fn := NewCache(CacheConfig{
		Store: &testStore{},
	})

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
	c.Next = func() error {
		c.CacheMaxAge(time.Minute)
		c.BodyBuffer = bytes.NewBufferString("hello world!")
		return nil
	}
	err = fn(c)
	assert.Nil(err)
	assert.Equal("hello world!", c.BodyBuffer.String())
	assert.Equal("fetch", c.GetHeader(HeaderXCache))

	// hit
	c = elton.NewContext(httptest.NewRecorder(), cacheableReq)
	c.Next = func() error {
		// 直接读取缓存，不再调用next
		return errors.New("abc")
	}
	err = fn(c)
	assert.Nil(err)
	assert.Equal("hello world!", c.BodyBuffer.String())
	assert.Equal("hit", c.GetHeader(HeaderXCache))
}
