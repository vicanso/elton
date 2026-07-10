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
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/hes"
)

const (
	// ErrRouterConcurrentLimiterCategory router concurrent limiter error category
	ErrRouterConcurrentLimiterCategory = "elton-router-concurrent-limiter"
)

var (
	ErrRequireLimiter = errors.New("require limiter")
)

type (
	// Config router concurrent limiter config
	RouterConcurrentLimiterConfig struct {
		Skipper elton.Skipper
		Limiter RouterConcurrencyLimiter
	}
	routerConcurrency struct {
		max     uint32
		current atomic.Uint32
	}
	// RouterConcurrencyLimiter limiter interface
	RouterConcurrencyLimiter interface {
		IncConcurrency(route string) (current uint32, max uint32)
		DecConcurrency(route string)
		GetConcurrency(route string) (current uint32)
	}
	// LocalLimiter local limiter
	LocalRouterConcurrencyLimiter struct {
		m map[string]*routerConcurrency
	}
)

// NewLocalRouterConcurrencyLimiter returns a new local limiter, it's useful for limit concurrency for process.
func NewLocalRouterConcurrencyLimiter(data map[string]uint32) *LocalRouterConcurrencyLimiter {
	m := make(map[string]*routerConcurrency, len(data))
	for route, max := range data {
		m[route] = &routerConcurrency{
			max: max,
		}
	}
	return &LocalRouterConcurrencyLimiter{
		m: m,
	}
}

// IncConcurrency inc 1
func (l *LocalRouterConcurrencyLimiter) IncConcurrency(key string) (uint32, uint32) {
	concur, ok := l.m[key]
	if !ok {
		return 0, 0
	}
	return concur.current.Add(1), concur.max
}

// DecConcurrency dec 1
func (l *LocalRouterConcurrencyLimiter) DecConcurrency(key string) {
	concur, ok := l.m[key]
	if !ok {
		return
	}
	// add ^uint32(0) means decrease 1
	concur.current.Add(^uint32(0))
}

// GetConcurrency value
func (l *LocalRouterConcurrencyLimiter) GetConcurrency(key string) uint32 {
	concur, ok := l.m[key]
	if !ok {
		return 0
	}
	return concur.current.Load()
}

func createRouterConcurrentLimiterError(current, max uint32) error {
	he := hes.New(fmt.Sprintf("too many request, current:%d, max:%d", current, max))
	he.Category = ErrRouterConcurrentLimiterCategory
	he.StatusCode = http.StatusTooManyRequests
	return he
}

// NewRouterConcurrentLimiter returns a router concurrent limiter middleware.
// It will throw panic if Limiter is nil.
func NewRouterConcurrentLimiter(config RouterConcurrentLimiterConfig) elton.Handler {
	skipper := getSkipper(config.Skipper)
	if config.Limiter == nil {
		panic(ErrRequireLimiter)
	}
	limiter := config.Limiter
	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		key := c.Request.Method + " " + c.Route
		current, max := limiter.IncConcurrency(key)
		defer limiter.DecConcurrency(key)
		if max != 0 && current > max {
			return createRouterConcurrentLimiterError(current, max)
		}
		return c.Next()
	}
}
