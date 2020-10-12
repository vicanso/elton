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

func TestResponderSkip(t *testing.T) {
	// skip处理
	assert := assert.New(t)
	c := elton.NewContext(nil, nil)
	c.Committed = true
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
}

func TestResponderResponseErr(t *testing.T) {
	// 响应出错时
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
}

func TestResponderResponseBuffer(t *testing.T) {
	// 响应数据已设置BodyBuffer
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
}

func newResponseServe(data interface{}, contentType string) *httptest.ResponseRecorder {
	e := elton.New()
	e.Use(NewDefaultResponder())
	e.GET("/", func(c *elton.Context) error {
		if contentType != "" {
			c.SetHeader(elton.HeaderContentType, contentType)
		}
		c.Body = data
		return nil
	})
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, httptest.NewRequest("GET", "/", nil))
	return resp
}

func TestResponderResponseInvalid(t *testing.T) {
	resp := newResponseServe(nil, "")
	checkResponse(t, resp, 500, "category=elton-responder, message=invalid response")
}

func TestResponderResponseString(t *testing.T) {
	resp := newResponseServe("abc", "")
	checkResponse(t, resp, 200, "abc")
	checkContentType(t, resp, "text/plain; charset=UTF-8")
}

func TestResponderResponseBytes(t *testing.T) {
	resp := newResponseServe([]byte("abc"), "")
	checkResponse(t, resp, 200, "abc")
	checkContentType(t, resp, elton.MIMEBinary)
}
func TestResponderResponseBytesWithContentType(t *testing.T) {
	conteType := "t"
	resp := newResponseServe([]byte("abc"), conteType)
	checkResponse(t, resp, 200, "abc")
	checkContentType(t, resp, conteType)
}

func TestResponderResponseStruct(t *testing.T) {
	type T struct {
		Name string `json:"name,omitempty"`
	}
	resp := newResponseServe(&T{
		Name: "tree.xie",
	}, "")
	checkResponse(t, resp, 200, `{"name":"tree.xie"}`)
	checkJSON(t, resp)
}

func TestResponderMarshalFail(t *testing.T) {
	assert := assert.New(t)
	resp := newResponseServe(func() {}, "")
	assert.Equal(500, resp.Code)
	assert.Equal("message=json: unsupported type: func()", resp.Body.String())
}

func TestResponderResponseReaderBody(t *testing.T) {
	assert := assert.New(t)
	resp := newResponseServe(bytes.NewReader([]byte("abcd")), "")
	assert.Equal(200, resp.Code)
	assert.Equal("abcd", resp.Body.String())
}

func TestResponderNoContent(t *testing.T) {
	assert := assert.New(t)
	e := elton.New()
	e.Use(NewDefaultResponder())
	e.GET("/", func(c *elton.Context) error {
		c.StatusCode = http.StatusNoContent
		return nil
	})
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, httptest.NewRequest("GET", "/", nil))
	assert.Equal(http.StatusNoContent, resp.Code)
}
