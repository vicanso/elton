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

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

const (
	// ErrRCLCategory router concurrent limiter error category
	ErrRCLCategory = "elton-router-concurrent-limiter"
)

var (
	ErrRCLRequireLimiter = errors.New("require limiter")
)

type (
	// Config router concurrent limiter config
	RCLConfig struct {
		Skipper elton.Skipper
		Limiter RCLLimiter
	}
	rclConcurrency struct {
		max     uint32
		current uint32
	}
	// RCLLimiter limiter interface
	RCLLimiter interface {
		IncConcurrency(route string) (current uint32, max uint32)
		DecConcurrency(route string)
		GetConcurrency(route string) (current uint32)
	}
	// LocalLimiter local limiter
	RCLLocalLimiter struct {
		m map[string]*rclConcurrency
	}
)

// NewLocalLimiter create a new limiter
func NewLocalLimiter(data map[string]uint32) *RCLLocalLimiter {
	m := make(map[string]*rclConcurrency, len(data))
	for route, max := range data {
		m[route] = &rclConcurrency{
			max:     max,
			current: 0,
		}
	}
	return &RCLLocalLimiter{
		m: m,
	}
}

// IncConcurrency concurrency inc
func (l *RCLLocalLimiter) IncConcurrency(key string) (current, max uint32) {
	concur, ok := l.m[key]
	if !ok {
		return 0, 0
	}
	v := atomic.AddUint32(&concur.current, 1)
	return v, concur.max
}

// DecConcurrency concurrency dec
func (l *RCLLocalLimiter) DecConcurrency(key string) {
	concur, ok := l.m[key]
	if !ok {
		return
	}
	atomic.AddUint32(&concur.current, ^uint32(0))
}

// GetConcurrency get concurrency
func (l *RCLLocalLimiter) GetConcurrency(key string) uint32 {
	concur, ok := l.m[key]
	if !ok {
		return 0
	}
	return atomic.LoadUint32(&concur.current)
}

func createRCLError(current, max uint32) error {
	he := hes.New(fmt.Sprintf("too many request, current:%d, max:%d", current, max))
	he.Category = ErrRCLCategory
	he.StatusCode = http.StatusTooManyRequests
	return he
}

// NewRCL create a router concurrent limiter middleware
func NewRCL(config RCLConfig) elton.Handler {
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	if config.Limiter == nil {
		panic(ErrRCLRequireLimiter)
	}
	limiter := config.Limiter
	return func(c *elton.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		key := c.Request.Method + " " + c.Route
		current, max := limiter.IncConcurrency(key)
		defer limiter.DecConcurrency(key)
		if max != 0 && current > max {
			err = createRCLError(current, max)
			return
		}
		return c.Next()
	}
}
