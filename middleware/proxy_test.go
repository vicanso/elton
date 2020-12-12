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
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

func TestNoTargetPanic(t *testing.T) {
	assert := assert.New(t)
	defer func() {
		r := recover()
		assert.Equal(ErrProxyNoTargetFunction, r.(error))
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
	assert.Equal(0, len(regs), "rewrite regexp map should be 0")

	regs, err = generateRewrites([]string{
		"/(d/:a",
	})
	assert.NotNil(err)
	assert.Equal(0, len(regs), "regexp map should be 0 when error occur")
}

func newServer() (net.Listener, *url.URL) {
	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		panic(err)
	}

	e := elton.New()

	e.GET("/", func(c *elton.Context) error {
		c.BodyBuffer = bytes.NewBufferString(c.Request.Host)
		return nil
	})
	go func() {
		_ = e.Server.Serve(l)
	}()
	time.Sleep(10 * time.Millisecond)
	target, _ := url.Parse("http://" + l.Addr().String())
	return l, target
}

func TestProxy(t *testing.T) {
	// 使用target picker方法来获取target
	l, target := newServer()
	defer l.Close()
	assert := assert.New(t)

	skipErr := errors.New("skip error")
	// next直接返回skip error，用于判断是否执行了next
	next := func() error {
		return skipErr
	}
	proxyDoneKey := "_proxyDone"

	tests := []struct {
		newContext func() *elton.Context
		fn         elton.Handler
		err        error
		result     *bytes.Buffer
		statusCode int
		target     string
		proxyDone  bool
	}{
		// target pick fail
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				return c
			},
			fn: NewProxy(ProxyConfig{
				TargetPicker: func(c *elton.Context) (*url.URL, ProxyDone, error) {
					return nil, nil, errors.New("abcd")
				},
				Host:      "www.baidu.com",
				Transport: &http.Transport{},
			}),
			err: errors.New("abcd"),
		},
		// no match target
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				return c
			},
			fn: NewProxy(ProxyConfig{
				TargetPicker: func(c *elton.Context) (*url.URL, ProxyDone, error) {
					return nil, nil, nil
				},
				Host:      "www.baidu.com",
				Transport: &http.Transport{},
			}),
			err: ErrProxyTargetIsNil,
		},
		// proxy request fail
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.Next = func() error {
					return nil
				}
				return c
			},
			fn: NewProxy(ProxyConfig{
				TargetPicker: func(c *elton.Context) (*url.URL, ProxyDone, error) {
					target, _ := url.Parse("https://127.0.0.1")
					return target, nil, nil
				},
				Transport: &http.Transport{},
			}),
			err: &hes.Error{
				Category:  ErrProxyCategory,
				Message:   "dial tcp 127.0.0.1:443: connect: connection refused",
				Exception: true,
			},
			target: "https://127.0.0.1",
		},
		// proxy target with done(success)
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.Next = next
				return c
			},
			// proxy中有done的处理，主要用于选择target时使用最少连接数的处理
			fn: NewProxy(ProxyConfig{
				Target:    target,
				Host:      "www.baidu.com",
				Transport: &http.Transport{},
				Done: func(c *elton.Context) {
					c.Set(proxyDoneKey, true)
				},
			}),
			target:     target.String(),
			result:     bytes.NewBufferString("www.baidu.com"),
			statusCode: 200,
			proxyDone:  true,
			err:        skipErr,
		},
		// normal(success)
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.Next = next
				return c
			},
			fn: NewProxy(ProxyConfig{
				TargetPicker: func(c *elton.Context) (*url.URL, ProxyDone, error) {
					return target, func(_ *elton.Context) {
						c.Set(proxyDoneKey, true)
					}, nil
				},
				Host:      "www.baidu.com",
				Transport: &http.Transport{},
			}),
			target:     target.String(),
			err:        skipErr,
			result:     bytes.NewBufferString("www.baidu.com"),
			statusCode: 200,
			proxyDone:  true,
		},
		// rewrite(success)
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "http://127.0.0.1/api/", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.Next = next
				return c
			},
			fn: NewProxy(ProxyConfig{
				Target:    target,
				Host:      "github.com",
				Transport: &http.Transport{},
				Rewrites: []string{
					"/api/*:/$1",
				},
			}),
			target:     target.String(),
			err:        skipErr,
			result:     bytes.NewBufferString("github.com"),
			statusCode: 200,
		},
	}
	for _, tt := range tests {
		c := tt.newContext()
		err := tt.fn(c)
		if err != nil || tt.err != nil {
			assert.Equal(tt.err.Error(), err.Error())
		}
		assert.Equal(tt.statusCode, c.StatusCode)
		assert.Equal(tt.target, c.GetString(ProxyTargetKey))
		assert.Equal(tt.proxyDone, c.GetBool(proxyDoneKey))
		assert.Equal(tt.result, c.BodyBuffer)
	}

}
