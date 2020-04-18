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

func checkResponse(t *testing.T, resp *httptest.ResponseRecorder, code int, data string) {
	assert := assert.New(t)
	assert.Equal(data, resp.Body.String())
	assert.Equal(code, resp.Code)
}

func checkJSON(t *testing.T, resp *httptest.ResponseRecorder) {
	assert := assert.New(t)
	assert.Equal(elton.MIMEApplicationJSON, resp.Header().Get(elton.HeaderContentType))
}

func checkContentType(t *testing.T, resp *httptest.ResponseRecorder, contentType string) {
	assert := assert.New(t)
	assert.Equal(contentType, resp.Header().Get(elton.HeaderContentType))
}

func TestResponder(t *testing.T) {
	m := NewResponder(ResponderConfig{})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)

	t.Run("skip", func(t *testing.T) {
		assert := assert.New(t)
		c := elton.NewContext(nil, nil)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		fn := NewResponder(ResponderConfig{
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
		customErr := errors.New("abcd")
		c := elton.NewContext(nil, nil)
		done := false
		c.Next = func() error {
			done = true
			return customErr
		}
		fn := NewDefaultResponder()
		err := fn(c)
		assert.Equal(customErr, err)
		assert.True(done)
	})

	t.Run("set BodyBuffer", func(t *testing.T) {
		assert := assert.New(t)
		c := elton.NewContext(nil, nil)
		done := false
		c.Next = func() error {
			c.BodyBuffer = bytes.NewBuffer([]byte(""))
			done = true
			return nil
		}
		fn := NewResponder(ResponderConfig{})
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
	})

	t.Run("invalid response", func(t *testing.T) {
		e := elton.New()
		e.Use(m)
		e.GET("/", func(c *elton.Context) error {
			return nil
		})
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		checkResponse(t, resp, 500, "category=elton-responder, message=invalid response")
	})

	t.Run("return string", func(t *testing.T) {
		e := elton.New()
		e.Use(m)
		e.GET("/", func(c *elton.Context) error {
			c.Body = "abc"
			return nil
		})
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		checkResponse(t, resp, 200, "abc")
		checkContentType(t, resp, "text/plain; charset=UTF-8")
	})

	t.Run("return bytes", func(t *testing.T) {
		e := elton.New()
		e.Use(m)
		e.GET("/", func(c *elton.Context) error {
			c.Body = []byte("abc")
			return nil
		})
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		checkResponse(t, resp, 200, "abc")
		checkContentType(t, resp, elton.MIMEBinary)
	})
	t.Run("return bytes(set content type)", func(t *testing.T) {
		e := elton.New()
		e.Use(m)
		e.GET("/", func(c *elton.Context) error {
			c.Body = []byte("abc")
			return nil
		})
		resp := httptest.NewRecorder()

		contentType := "abc"
		resp.Header().Set(elton.HeaderContentType, contentType)
		e.ServeHTTP(resp, req)
		checkResponse(t, resp, 200, "abc")
		checkContentType(t, resp, contentType)
	})

	t.Run("return struct", func(t *testing.T) {
		type T struct {
			Name string `json:"name,omitempty"`
		}
		e := elton.New()
		e.Use(m)
		e.GET("/", func(c *elton.Context) error {
			c.Created(&T{
				Name: "tree.xie",
			})
			return nil
		})
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		checkResponse(t, resp, 201, `{"name":"tree.xie"}`)
		checkJSON(t, resp)
	})

	t.Run("json marshal fail", func(t *testing.T) {
		assert := assert.New(t)
		e := elton.New()
		e.Use(m)
		e.GET("/", func(c *elton.Context) error {
			c.Body = func() {}
			return nil
		})
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(500, resp.Code)
		assert.Equal("message=json: unsupported type: func()", resp.Body.String())
	})

	t.Run("reader body", func(t *testing.T) {
		assert := assert.New(t)
		e := elton.New()
		e.Use(m)
		e.GET("/", func(c *elton.Context) error {
			c.Body = bytes.NewReader([]byte("abcd"))
			return nil
		})
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(200, resp.Code)
		assert.Equal("abcd", resp.Body.String())
	})

	t.Run("no content", func(t *testing.T) {
		assert := assert.New(t)
		e := elton.New()
		e.Use(m)
		e.GET("/", func(c *elton.Context) error {
			c.StatusCode = http.StatusNoContent
			return nil
		})
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(http.StatusNoContent, resp.Code)
	})

	t.Run("accepted(202)", func(t *testing.T) {
		assert := assert.New(t)
		e := elton.New()
		e.Use(m)
		e.GET("/", func(c *elton.Context) error {
			c.StatusCode = http.StatusAccepted
			return nil
		})
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Empty(resp.Body)
		assert.Equal(http.StatusAccepted, resp.Code)
	})
}
