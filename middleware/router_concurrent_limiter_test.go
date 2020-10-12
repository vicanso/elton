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
	"net/http/httptest"
	"sync/atomic"
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

func TestRouterConcurrentLimiterSkip(t *testing.T) {
	assert := assert.New(t)
	req := httptest.NewRequest("GET", "/", nil)
	c := elton.NewContext(nil, req)
	c.Committed = true
	done := false
	c.Next = func() error {
		done = true
		return nil
	}
	err := newLimiterMiddleware()(c)
	assert.Nil(err)
	assert.True(done)
}

func TestRouterConcurrentLimiter(t *testing.T) {
	assert := assert.New(t)
	req := httptest.NewRequest("GET", "/books/1", nil)
	c := elton.NewContext(nil, req)
	c.Route = "/books/:id"
	var count int32
	max := 10
	c.Next = func() error {
		atomic.AddInt32(&count, 1)
		return nil
	}

	fn := newLimiterMiddleware()
	for index := 0; index < max; index++ {
		err := fn(c)
		assert.Nil(err)
	}
	assert.Equal(int32(max), count)
}

func TestRouterConcurrentLimiterOverLimit(t *testing.T) {
	assert := assert.New(t)
	fn := newLimiterMiddleware()

	req := httptest.NewRequest("POST", "/users/login", nil)
	c := elton.NewContext(nil, req)
	c.Route = "/users/login"
	c.Next = func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	done := make(chan bool)
	go func() {
		time.Sleep(2 * time.Millisecond)
		err := fn(c)
		assert.NotNil(err)
		assert.Equal("category=elton-router-concurrent-limiter, message=too many request, current:2, max:1", err.Error())
		done <- true
	}()
	err := fn(c)
	assert.Nil(err)
	<-done
}
