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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestRCLLimiter(t *testing.T) {

	assert := assert.New(t)
	limiter := NewLocalLimiter(map[string]uint32{
		"/users/login": 10,
		"/books/:id":   100,
	})

	cur, max := limiter.IncConcurrency("/not-match-route")
	assert.Equal(uint32(0), max)
	assert.Equal(uint32(0), cur)

	cur, max = limiter.IncConcurrency("/users/login")
	assert.Equal(uint32(10), max)
	assert.Equal(uint32(1), cur)

	limiter.DecConcurrency("/not-match-route")
	assert.Equal(uint32(0), limiter.GetConcurrency("/not-match-route"))

	limiter.DecConcurrency("/users/login")
	assert.Equal(uint32(0), limiter.GetConcurrency("/users/login"))
}

func TestRCLNoLimiterPanic(t *testing.T) {
	assert := assert.New(t)
	defer func() {
		r := recover()
		assert.NotNil(r)
		assert.Equal(r.(error), ErrRCLRequireLimiter)
	}()

	NewRCL(RCLConfig{})
}

func newLimiterMiddleware() elton.Handler {
	limiter := NewLocalLimiter(map[string]uint32{
		"POST /users/login": 1,
		"GET /books/:id":    100,
	})
	return NewRCL(RCLConfig{
		Limiter: limiter,
	})
}

func TestRouterConcurrentLimiter(t *testing.T) {
	assert := assert.New(t)
	skipErr := errors.New("skip error")
	// next直接返回skip error，用于判断是否执行了next
	next := func() error {
		return skipErr
	}

	defaultLimiter := NewRCL(RCLConfig{
		Limiter: NewLocalLimiter(map[string]uint32{
			"POST /users/login": 1,
			"GET /books/:id":    100,
		}),
	})

	tests := []struct {
		newContext func() *elton.Context
		err        error
	}{
		// skip
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/", nil)
				c := elton.NewContext(nil, req)
				c.Committed = true
				c.Next = next
				return c
			},
			err: skipErr,
		},
		// over limit
		{
			newContext: func() *elton.Context {
				go func() {
					req := httptest.NewRequest("POST", "/users/login", nil)
					c := elton.NewContext(nil, req)
					c.Route = "/users/login"
					c.Next = func() error {
						// 该请求在处理，但延时完成
						time.Sleep(10 * time.Millisecond)
						return nil
					}
					_ = defaultLimiter(c)
				}()
				// 延时，保证第一个请求已进入
				time.Sleep(2 * time.Millisecond)
				req := httptest.NewRequest("POST", "/users/login", nil)
				c := elton.NewContext(nil, req)
				c.Route = "/users/login"
				c.Next = func() error {
					time.Sleep(10 * time.Millisecond)
					return nil
				}
				return c
			},
			err: createRCLError(2, 1),
		},
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/books/1", nil)
				c := elton.NewContext(nil, req)
				c.Route = "/books/:id"
				c.Next = next
				return c
			},
			err: skipErr,
		},
	}

	for _, tt := range tests {
		c := tt.newContext()
		err := defaultLimiter(c)
		assert.Equal(tt.err, err)
	}
}
