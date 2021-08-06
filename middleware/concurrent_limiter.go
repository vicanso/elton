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
	"strings"
	"sync/atomic"

	"github.com/tidwall/gjson"
	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

var (
	// ErrSubmitTooFrequently submit too frequently
	ErrSubmitTooFrequently = &hes.Error{
		StatusCode: http.StatusBadRequest,
		Message:    "submit too frequently",
		Category:   ErrConcurrentLimiterCategory,
	}
	// ErrTooManyRequests too many request
	ErrTooManyRequests = &hes.Error{
		StatusCode: http.StatusTooManyRequests,
		Message:    "too many Requests",
		Category:   ErrConcurrentLimiterCategory,
	}
	// ErrNotAllowEmpty not allow empty
	ErrNotAllowEmpty = &hes.Error{
		StatusCode: http.StatusBadRequest,
		Message:    "empty value is not allowed",
		Category:   ErrConcurrentLimiterCategory,
	}
	ErrRequireLockFunction = errors.New("require lock function")
)

const (
	ipKey     = ":ip"
	headerKey = "h:"
	queryKey  = "q:"
	paramKey  = "p:"
	// ErrConcurrentLimiterCategory concurrent limiter error category
	ErrConcurrentLimiterCategory = "elton-concurrent-limiter"
)

type (
	// ConcurrentLimiterLock lock the key
	ConcurrentLimiterLock func(string, *elton.Context) (bool, func(), error)
	// Config concurrent limiter config
	ConcurrentLimiterConfig struct {
		// Keys keys for generate lock id
		Keys []string
		// Lock lock function
		Lock    ConcurrentLimiterLock
		Skipper elton.Skipper
		// NotAllowEmpty if value is empty, will return error
		NotAllowEmpty bool
	}
	// concurrentLimiterKeyInfo the concurrent key's info
	concurrentLimiterKeyInfo struct {
		Name   string
		Params bool
		Query  bool
		Header bool
		Body   bool
		IP     bool
	}
	// GlobalConcurrentLimiterConfig
	GlobalConcurrentLimiterConfig struct {
		Skipper elton.Skipper
		Max     uint32
	}
)

// NewConcurrentLimiter returns a new concurrent limiter middleware.
// It will throw a panic if Lock function is nil.
func NewConcurrentLimiter(config ConcurrentLimiterConfig) elton.Handler {

	if config.Lock == nil {
		panic(ErrRequireLockFunction)
	}
	keys := make([]*concurrentLimiterKeyInfo, 0)
	// 根据配置生成key的处理
	for _, key := range config.Keys {
		if key == ipKey {
			keys = append(keys, &concurrentLimiterKeyInfo{
				IP: true,
			})
			continue
		}
		if strings.HasPrefix(key, headerKey) {
			keys = append(keys, &concurrentLimiterKeyInfo{
				Name:   key[2:],
				Header: true,
			})
			continue
		}
		if strings.HasPrefix(key, queryKey) {
			keys = append(keys, &concurrentLimiterKeyInfo{
				Name:  key[2:],
				Query: true,
			})
			continue
		}
		if strings.HasPrefix(key, paramKey) {
			keys = append(keys, &concurrentLimiterKeyInfo{
				Name:   key[2:],
				Params: true,
			})
			continue
		}
		keys = append(keys, &concurrentLimiterKeyInfo{
			Name: key,
			Body: true,
		})
	}
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	keyLength := len(keys)
	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		sb := new(strings.Builder)
		// 先申请假定每个value的长度
		sb.Grow(8 * keyLength)
		// 获取 lock 的key
		for i, key := range keys {
			v := ""
			name := key.Name
			if key.IP {
				v = c.RealIP()
			} else if key.Header {
				v = c.GetRequestHeader(name)
			} else if key.Query {
				v = c.QueryParam(name)
			} else if key.Params {
				v = c.Param(name)
			} else {
				v = gjson.GetBytes(c.RequestBody, name).String()
			}
			if config.NotAllowEmpty && len(v) == 0 {
				return ErrNotAllowEmpty
			}
			sb.WriteString(v)
			if i < keyLength-1 {
				sb.WriteRune(',')
			}
		}
		lockKey := sb.String()

		success, unlock, err := config.Lock(lockKey, c)
		if err != nil {
			return hes.Wrap(err)
		}
		if !success {
			return ErrSubmitTooFrequently
		}

		if unlock != nil {
			defer unlock()
		}

		return c.Next()
	}
}

// NewGlobalConcurrentLimiter returns a new global concurrent limiter,
// it use for global processing request limit.
func NewGlobalConcurrentLimiter(config GlobalConcurrentLimiterConfig) elton.Handler {
	var count uint32
	return func(c *elton.Context) error {
		value := atomic.AddUint32(&count, 1)
		defer atomic.AddUint32(&count, ^uint32(0))
		if value >= config.Max {
			return ErrTooManyRequests
		}
		return c.Next()
	}
}
