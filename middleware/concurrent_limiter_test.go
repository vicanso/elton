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
	"time"

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

func TestConcurrentLimiterNotAllowEmpty(t *testing.T) {
	// 设置不允许值为空的
	assert := assert.New(t)
	fn := NewConcurrentLimiter(ConcurrentLimiterConfig{
		NotAllowEmpty: true,
		Keys: []string{
			"p:id",
		},
		Lock: func(key string, c *elton.Context) (success bool, unlock func(), err error) {
			return
		},
	})
	c := elton.NewContext(nil, httptest.NewRequest("GET", "/", nil))
	err := fn(c)
	assert.Equal(ErrNotAllowEmpty, err)
}

func TestConcurrentLimiterLockError(t *testing.T) {
	// 当lock出错时
	assert := assert.New(t)
	fn := NewConcurrentLimiter(ConcurrentLimiterConfig{
		Keys: []string{
			"p:id",
		},
		Lock: func(key string, c *elton.Context) (success bool, unlock func(), err error) {
			return false, nil, errors.New("lock error")
		},
	})
	c := elton.NewContext(nil, httptest.NewRequest("GET", "/", nil))
	err := fn(c)
	he, ok := err.(*hes.Error)
	assert.True(ok)
	assert.Equal("message=lock error", he.Error())
}

func TestConcurrentLimiter(t *testing.T) {
	m := new(sync.Map)
	fn := NewConcurrentLimiter(ConcurrentLimiterConfig{
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

	req := httptest.NewRequest("POST", "/users/login?type=1", nil)
	resp := httptest.NewRecorder()
	c := elton.NewContext(resp, req)
	req.Header.Set("X-Token", "xyz")
	c.RequestBody = []byte(`{
		"account": "tree.xie"
	}`)
	c.Params = new(elton.RouteParams)
	c.Params.Add("id", "123")

	t.Run("first", func(t *testing.T) {
		assert := assert.New(t)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
	})

	t.Run("too frequently", func(t *testing.T) {
		assert := assert.New(t)
		done := false
		c.Next = func() error {
			time.Sleep(100 * time.Millisecond)
			done = true
			return nil
		}
		go func() {
			time.Sleep(10 * time.Millisecond)
			e := fn(c)
			assert.Equal(e.Error(), "category=elton-concurrent-limiter, message=submit too frequently")
		}()
		err := fn(c)
		// 登录限制,192.0.2.1,xyz,1,123,tree.xie
		assert.Nil(err)
		assert.True(done)
	})
}

func TestGlobalConcurrentLimiter(t *testing.T) {
	assert := assert.New(t)
	fn := NewGlobalConcurrentLimiter(GlobalConcurrentLimiterConfig{
		Max: 1,
	})
	req := httptest.NewRequest("POST", "/users/login?type=1", nil)
	resp := httptest.NewRecorder()
	c := elton.NewContext(resp, req)
	err := fn(c)
	assert.Equal(ErrTooManyRequests, err)

	fn = NewGlobalConcurrentLimiter(GlobalConcurrentLimiterConfig{
		Max: 2,
	})
	done := false
	c.Next = func() error {
		done = true
		return nil
	}
	err = fn(c)
	assert.Nil(err)
	assert.True(done)
}
