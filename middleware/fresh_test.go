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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestFresh(t *testing.T) {
	fn := NewDefaultFresh()
	modifiedAt := "Tue, 25 Dec 2018 00:02:22 GMT"
	t.Run("skip", func(t *testing.T) {
		assert := assert.New(t)
		c := elton.NewContext(nil, nil)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		fn := NewFresh(FreshConfig{
			Skipper: func(c *elton.Context) bool {
				return true
			},
		})
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
	})

	t.Run("return error", func(t *testing.T) {
		assert := assert.New(t)
		c := elton.NewContext(nil, nil)
		customErr := errors.New("abccd")
		c.Next = func() error {
			return customErr
		}
		fn := NewFresh(FreshConfig{})
		err := fn(c)
		assert.Equal(err, customErr, "custom error should be return")
	})

	t.Run("not modified", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(elton.HeaderIfModifiedSince, modifiedAt)
		resp := httptest.NewRecorder()
		resp.Header().Set(elton.HeaderLastModified, modifiedAt)

		c := elton.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			c.Body = map[string]string{
				"name": "tree.xie",
			}
			c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.True(done)

		assert.Equal(c.StatusCode, 304, "status code should be 304")
		assert.Nil(c.Body, "body should be nil")
		assert.Nil(c.BodyBuffer, "body buffer should be nil")
	})

	t.Run("no body", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(elton.HeaderIfModifiedSince, modifiedAt)
		resp := httptest.NewRecorder()
		resp.Header().Set(elton.HeaderLastModified, modifiedAt)
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		c.NoContent()
		err := fn(c)
		assert.Nil(err)
		assert.Equal(c.StatusCode, 204, "no body should be passed by fresh")
	})

	t.Run("post method", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("POST", "/users/me", nil)
		req.Header.Set(elton.HeaderIfModifiedSince, modifiedAt)
		resp := httptest.NewRecorder()
		resp.Header().Set(elton.HeaderLastModified, modifiedAt)

		c := elton.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			c.StatusCode = http.StatusOK
			c.Body = map[string]string{
				"name": "tree.xie",
			}
			c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.True(done)

		assert.Equal(c.StatusCode, 200, "post requset should be passed by fresh")
		assert.NotNil(c.Body, "post requset should be passed by fresh")
		assert.NotNil(c.BodyBuffer, "post requset should be passed by fresh")
	})

	t.Run("error response", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(elton.HeaderIfModifiedSince, modifiedAt)
		resp := httptest.NewRecorder()
		resp.Header().Set(elton.HeaderLastModified, modifiedAt)

		c := elton.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			c.StatusCode = http.StatusBadRequest
			c.Body = map[string]string{
				"name": "tree.xie",
			}
			c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.True(done)

		assert.Equal(c.StatusCode, http.StatusBadRequest, "error response should be passed by fresh")
		assert.NotNil(c.Body, "error response should be passed by fresh")
		assert.NotNil(c.BodyBuffer, "error response should be passed by fresh")
	})
}
