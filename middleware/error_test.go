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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestSkipAndNoError(t *testing.T) {
	fn := NewDefaultError()
	t.Run("skip", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "/users/me", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Committed = true
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.Nil(c.BodyBuffer)
	})

	t.Run("no error", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "/users/me", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.Nil(c.BodyBuffer)
	})
}

func TestErrorHandler(t *testing.T) {
	t.Run("json type", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewDefaultError()
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set("Accept", "application/json, text/plain, */*")
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return errors.New("abcd")
		}
		c.CacheMaxAge("5m")
		err := fn(c)
		assert.Nil(err)
		assert.Equal("public, max-age=300", c.GetHeader(elton.HeaderCacheControl))
		assert.True(strings.HasSuffix(c.BodyBuffer.String(), `"statusCode":500,"category":"elton-error","message":"abcd","exception":true}`))
		assert.Equal("application/json; charset=UTF-8", c.GetHeader(elton.HeaderContentType))
	})

	t.Run("text type", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewError(ErrorConfig{
			ResponseType: "text",
		})
		req := httptest.NewRequest("GET", "/users/me", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return errors.New("abcd")
		}
		c.CacheMaxAge("5m")
		err := fn(c)
		assert.Nil(err)
		assert.Equal("public, max-age=300", c.GetHeader(elton.HeaderCacheControl))
		ct := c.GetHeader(elton.HeaderContentType)
		assert.Equal("category=elton-error, message=abcd", c.BodyBuffer.String())
		assert.Equal("text/plain; charset=UTF-8", ct)
	})
}
