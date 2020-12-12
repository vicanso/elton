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
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

func TestNoLockFunction(t *testing.T) {
	assert := assert.New(t)
	defer func() {
		r := recover()
		assert.Equal(r.(error), ErrRequireLockFunction)
	}()

	NewConcurrentLimiter(ConcurrentLimiterConfig{})
}

func TestConcurrentLimiter(t *testing.T) {
	assert := assert.New(t)
	m := new(sync.Map)
	concurrentLimiter := NewConcurrentLimiter(ConcurrentLimiterConfig{
		Keys: []string{
			":ip",
			"h:X-Token",
			"q:type",
			"p:id",
			"account",
		},
		Lock: func(key string, c *elton.Context) (success bool, unlock func(), err error) {
			if key != "192.0.2.1,xyz,1,123,tree.xie" {
				err = errors.New("key is invalid")
				return
			}
			_, loaded := m.LoadOrStore(key, 1)
			// 如果已存在，则获取锁失败
			if loaded {
				return
			}
			success = true
			// 删除锁
			unlock = func() {
				m.Delete(key)
			}
			return
		},
	})

	skipErr := errors.New("skip error")
	// next直接返回skip error，用于判断是否执行了next
	next := func() error {
		return skipErr
	}
	tests := []struct {
		newContext func() *elton.Context
		fn         elton.Handler
		err        error
	}{
		// not allow empty
		{
			newContext: func() *elton.Context {
				return elton.NewContext(nil, httptest.NewRequest("GET", "/", nil))
			},
			fn: NewConcurrentLimiter(ConcurrentLimiterConfig{
				NotAllowEmpty: true,
				Keys: []string{
					"p:id",
				},
				Lock: func(key string, c *elton.Context) (success bool, unlock func(), err error) {
					return
				},
			}),
			err: ErrNotAllowEmpty,
		},
		// lock fail
		{
			newContext: func() *elton.Context {
				return elton.NewContext(nil, httptest.NewRequest("GET", "/", nil))
			},
			fn: NewConcurrentLimiter(ConcurrentLimiterConfig{
				Keys: []string{
					"p:id",
				},
				Lock: func(key string, c *elton.Context) (success bool, unlock func(), err error) {
					return false, nil, errors.New("lock error")
				},
			}),
			err: hes.NewWithError(errors.New("lock error")),
		},
		// global concurrency limit 1(fail)
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("POST", "/users/login?type=1", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				return c
			},
			fn: NewGlobalConcurrentLimiter(GlobalConcurrentLimiterConfig{
				Max: 1,
			}),
			err: ErrTooManyRequests,
		},
		// global concurrency limit 2(success)
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("POST", "/users/login?type=1", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.Next = next
				return c
			},
			fn: NewGlobalConcurrentLimiter(GlobalConcurrentLimiterConfig{
				Max: 2,
			}),
			err: skipErr,
		},
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("POST", "/users/login?type=1", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				req.Header.Set("X-Token", "xyz")
				c.RequestBody = []byte(`{
					"account": "tree.xie"
				}`)
				c.Params = new(elton.RouteParams)
				c.Params.Add("id", "123")
				c.Next = next
				return c
			},
			fn:  concurrentLimiter,
			err: skipErr,
		},
	}

	for _, tt := range tests {
		c := tt.newContext()
		err := tt.fn(c)
		assert.Equal(tt.err, err)
	}
}
