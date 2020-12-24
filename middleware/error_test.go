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
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestErrorHandler(t *testing.T) {
	assert := assert.New(t)
	defaultErrorHandler := NewDefaultError()
	skipErr := errors.New("skip error")
	// next直接返回skip error，用于判断是否执行了next
	next := func() error {
		return skipErr
	}

	tests := []struct {
		newContext   func() *elton.Context
		fn           elton.Handler
		result       *bytes.Buffer
		cacheControl string
		contentType  string
		err          error
	}{
		// skip
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/users/me", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.Committed = true
				c.Next = next
				return c
			},
			fn:  defaultErrorHandler,
			err: skipErr,
		},
		// no error
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/users/me", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.Next = func() error {
					return nil
				}
				return c
			},
			fn: defaultErrorHandler,
		},
		// error(json)
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/users/me", nil)
				req.Header.Set("Accept", "application/json, text/plain, */*")
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.Next = func() error {
					return errors.New("abcd")
				}
				c.CacheMaxAge(5 * time.Minute)
				return c
			},
			fn:           defaultErrorHandler,
			result:       bytes.NewBufferString(`{"statusCode":500,"category":"elton-error","message":"abcd","exception":true}`),
			cacheControl: "public, max-age=300",
			contentType:  "application/json; charset=utf-8",
		},
		// error(text)
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/users/me", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.Next = func() error {
					return errors.New("abcd")
				}
				c.CacheMaxAge(5 * time.Minute)
				return c
			},
			fn:           defaultErrorHandler,
			result:       bytes.NewBufferString(`category=elton-error, message=abcd`),
			cacheControl: "public, max-age=300",
			contentType:  "text/plain; charset=utf-8",
		},
	}

	for _, tt := range tests {
		c := tt.newContext()
		err := tt.fn(c)
		assert.Equal(tt.err, err)
		assert.Equal(tt.result, c.BodyBuffer)
		assert.Equal(tt.cacheControl, c.GetHeader(elton.HeaderCacheControl))
		assert.Equal(tt.contentType, c.GetHeader(elton.HeaderContentType))
	}
}
