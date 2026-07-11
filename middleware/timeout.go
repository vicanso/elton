// MIT License

// Copyright (c) 2026 Tree Xie

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
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/hes"
)

const (
	// ErrTimeoutCategory timeout error category
	ErrTimeoutCategory = "elton-timeout"
)

// TimeoutConfig timeout middleware config
type TimeoutConfig struct {
	// Timeout request processing deadline; must be > 0
	Timeout time.Duration
	// Skipper skipper function
	Skipper elton.Skipper
	// Error custom timeout error; default is 504 Gateway Timeout
	Error error
}

// ErrRequestTimeout default timeout error (HTTP 504)
var ErrRequestTimeout = &hes.Error{
	StatusCode: http.StatusGatewayTimeout,
	Message:    "request timeout",
	Category:   ErrTimeoutCategory,
}

// NewTimeout returns a middleware that attaches a deadline to the request context.
// Handlers should honor c.Context() (e.g. pass to http clients / DB).
// If the deadline is exceeded when Next returns (or Next returns ctx deadline error),
// the configured timeout error is returned (default 504).
//
// Note: this does not hard-abort an already-running handler; it relies on context
// cancellation. Prefer placing timeout outside of long non-cancellable work.
func NewTimeout(config TimeoutConfig) elton.Handler {
	if config.Timeout <= 0 {
		panic(errors.New("timeout must be greater than 0"))
	}
	skipper := getSkipper(config.Skipper)
	timeoutErr := config.Error
	if timeoutErr == nil {
		timeoutErr = ErrRequestTimeout
	}
	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		ctx, cancel := context.WithTimeout(c.Context(), config.Timeout)
		defer cancel()
		c.WithContext(ctx)

		err := c.Next()
		if ctx.Err() == context.DeadlineExceeded {
			if err == nil ||
				errors.Is(err, context.DeadlineExceeded) ||
				errors.Is(err, context.Canceled) {
				return timeoutErr
			}
		}
		return err
	}
}

// NewDefaultTimeout returns timeout middleware with the given duration.
func NewDefaultTimeout(d time.Duration) elton.Handler {
	return NewTimeout(TimeoutConfig{Timeout: d})
}
