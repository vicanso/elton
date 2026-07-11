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
	"crypto/rand"
	"encoding/hex"

	"github.com/vicanso/elton/v2"
)

const (
	// HeaderXRequestID default request id header
	HeaderXRequestID = "X-Request-Id"
	// ContextKeyRequestID context store key for request id
	ContextKeyRequestID = "requestId"
)

// RequestIDConfig request id middleware config
type RequestIDConfig struct {
	// Skipper skipper function
	Skipper elton.Skipper
	// Header request/response header name; default X-Request-Id
	Header string
	// ContextKey c.Set key; default "requestId"
	ContextKey string
	// Generator custom id generator; default 16-byte hex
	Generator func() string
	// SetResponseHeader whether write id to response header; default true
	// Set to false if only storing in context / request header is needed.
	// Zero value is true when using NewDefaultRequestID; for NewRequestID, nil means true via pointer? Use bool with default true by checking via separate field.
	// Simpler: always set response header unless DisableResponseHeader is true.
	DisableResponseHeader bool
}

// NewDefaultRequestID returns request-id middleware with defaults.
func NewDefaultRequestID() elton.Handler {
	return NewRequestID(RequestIDConfig{})
}

// NewRequestID returns a middleware that ensures each request has an id:
// reuse inbound header if present, otherwise generate one; store on context
// and (by default) echo on the response header.
func NewRequestID(config RequestIDConfig) elton.Handler {
	skipper := getSkipper(config.Skipper)
	header := config.Header
	if header == "" {
		header = HeaderXRequestID
	}
	ctxKey := config.ContextKey
	if ctxKey == "" {
		ctxKey = ContextKeyRequestID
	}
	gen := config.Generator
	if gen == nil {
		gen = defaultRequestID
	}
	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		id := c.GetRequestHeader(header)
		if id == "" {
			id = gen()
			// 便于下游 logger {>X-Request-Id} 读到
			c.SetRequestHeader(header, id)
		}
		c.Set(ctxKey, id)
		// 与 elton.GenerateID 对齐时可选：不覆盖已有 ID
		if c.ID == "" {
			c.ID = id
		}
		if !config.DisableResponseHeader {
			c.SetHeader(header, id)
		}
		return c.Next()
	}
}

// GetRequestID returns the request id from context store (default key).
func GetRequestID(c *elton.Context) string {
	return elton.GetContextValue[string](c, ContextKeyRequestID)
}

func defaultRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// 极低概率：退化为固定长度零串仍可区分为空
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b[:])
}
