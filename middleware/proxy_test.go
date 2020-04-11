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
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestNoTargetPanic(t *testing.T) {
	assert := assert.New(t)
	defer func() {
		r := recover()
		assert.Equal(r.(error), ErrProxyNoTargetFunction)
	}()
	NewProxy(ProxyConfig{})
}

func TestInvalidRewrite(t *testing.T) {
	assert := assert.New(t)
	defer func() {
		r := recover()
		assert.NotNil(r.(error))
	}()
	target, _ := url.Parse("https://github.com")
	NewProxy(ProxyConfig{
		Target: target,
		Rewrites: []string{
			"/(d/:a",
		},
	})
}

func TestGenerateRewrites(t *testing.T) {
	assert := assert.New(t)
	regs, err := generateRewrites([]string{
		"a:b:c",
	})
	assert.Nil(err)
	assert.Equal(len(regs), 0, "rewrite regexp map should be 0")

	regs, err = generateRewrites([]string{
		"/(d/:a",
	})
	assert.NotNil(err)
	assert.Equal(len(regs), 0, "regexp map should be 0 when error occur")
}

func TestProxy(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		assert := assert.New(t)
		target, _ := url.Parse("https://github.com")
		config := ProxyConfig{
			Target:    target,
			Host:      "github.com",
			Transport: &http.Transport{},
			Rewrites: []string{
				"/api/*:/$1",
			},
		}
		fn := NewProxy(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/api/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		originalPath := req.URL.Path
		originalHost := req.Host
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.Equal(c.GetHeader("Content-Encoding"), "gzip")
		assert.Equal(c.Request.URL.Path, originalPath)
		assert.Equal(req.Host, originalHost)
		assert.True(done)
		assert.Equal(c.StatusCode, http.StatusOK)
	})

	t.Run("target picker", func(t *testing.T) {
		assert := assert.New(t)
		target, _ := url.Parse("https://www.baidu.com")
		callBackDone := false
		config := ProxyConfig{
			TargetPicker: func(c *elton.Context) (*url.URL, ProxyDone, error) {
				return target, func(_ *elton.Context) {
					callBackDone = true
				}, nil
			},
			Host:      "www.baidu.com",
			Transport: &http.Transport{},
		}
		fn := NewProxy(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
		assert.True(callBackDone)
		assert.Equal(c.StatusCode, http.StatusOK)
	})

	t.Run("target picker error", func(t *testing.T) {
		assert := assert.New(t)
		config := ProxyConfig{
			TargetPicker: func(c *elton.Context) (*url.URL, ProxyDone, error) {
				return nil, nil, errors.New("abcd")
			},
			Host:      "www.baidu.com",
			Transport: &http.Transport{},
		}
		fn := NewProxy(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		err := fn(c)
		assert.Equal(err.Error(), "abcd")
	})

	t.Run("no target", func(t *testing.T) {
		assert := assert.New(t)
		config := ProxyConfig{
			TargetPicker: func(c *elton.Context) (*url.URL, ProxyDone, error) {
				return nil, nil, nil
			},
			Host:      "www.baidu.com",
			Transport: &http.Transport{},
		}
		fn := NewProxy(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		err := fn(c)
		assert.Equal(err.Error(), "category=elton-proxy, message=target can not be nil")
	})

	t.Run("proxy error", func(t *testing.T) {
		assert := assert.New(t)
		target, _ := url.Parse("https://a")
		config := ProxyConfig{
			TargetPicker: func(c *elton.Context) (*url.URL, ProxyDone, error) {
				return target, nil, nil
			},
			Transport: &http.Transport{},
		}
		fn := NewProxy(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.NotNil(err)
	})

	t.Run("proxy done", func(t *testing.T) {
		assert := assert.New(t)
		target, _ := url.Parse("https://www.baidu.com")
		done := false
		config := ProxyConfig{
			Target:    target,
			Host:      "www.baidu.com",
			Transport: &http.Transport{},
			Done: func(_ *elton.Context) {
				done = true
			},
		}
		fn := NewProxy(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
	})
}
