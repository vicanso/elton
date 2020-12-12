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
	"github.com/vicanso/hes"
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
	assert := assert.New(t)
	skipErr := errors.New("skip error")
	// next直接返回skip error，用于判断是否执行了next
	next := func() error {
		return skipErr
	}

	defaultResponder := NewDefaultResponder()

	tests := []struct {
		newContext  func() *elton.Context
		fn          elton.Handler
		err         error
		result      *bytes.Buffer
		statusCode  int
		contentType string
	}{
		// skip
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), nil)
				c.Committed = true
				c.Next = next
				return c
			},
			err: skipErr,
		},
		// response error
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), nil)
				c.Next = next
				return c
			},
			err: skipErr,
		},
		// already set response
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), nil)
				c.Next = func() error {
					c.BodyBuffer = bytes.NewBuffer([]byte("abc"))
					return nil
				}
				return c
			},
			result: bytes.NewBuffer([]byte("abc")),
		},
		// response invalid
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), nil)
				c.Next = func() error {
					return nil
				}
				return c
			},
			err: ErrInvalidResponse,
		},
		// marshal fail
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), nil)
				c.Body = func() {}
				c.Next = func() error {
					return nil
				}
				return c
			},
			err: &hes.Error{
				Message:   "json: unsupported type: func()",
				Category:  ErrResponderCategory,
				Exception: true,
			},
		},
		// response string
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), nil)
				c.Body = "abc"
				c.Next = func() error {
					return nil
				}
				return c
			},
			result:      bytes.NewBufferString("abc"),
			statusCode:  200,
			contentType: "text/plain; charset=UTF-8",
		},
		// response byte
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), nil)
				c.Body = []byte("abc")
				c.Next = func() error {
					return nil
				}
				return c
			},
			result:      bytes.NewBufferString("abc"),
			statusCode:  200,
			contentType: "application/octet-stream",
		},
		// response with custom content type
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), nil)
				c.Body = []byte("abc")
				c.SetHeader(elton.HeaderContentType, "t")
				c.Next = func() error {
					return nil
				}
				return c
			},
			result:      bytes.NewBufferString("abc"),
			statusCode:  200,
			contentType: "t",
		},
		// response struct
		{
			newContext: func() *elton.Context {
				type T struct {
					Name string `json:"name,omitempty"`
				}
				c := elton.NewContext(httptest.NewRecorder(), nil)
				c.Body = &T{
					Name: "tree.xie",
				}
				c.Next = func() error {
					return nil
				}
				return c
			},
			result:      bytes.NewBufferString(`{"name":"tree.xie"}`),
			statusCode:  200,
			contentType: "application/json; charset=UTF-8",
		},
		// response reader
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), nil)
				c.Body = bytes.NewReader([]byte("abcd"))
				c.Next = func() error {
					return nil
				}
				return c
			},
		},
		// no content
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), nil)
				c.StatusCode = http.StatusNoContent
				c.Next = func() error {
					return nil
				}
				return c
			},
			statusCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		c := tt.newContext()
		err := defaultResponder(c)
		if err != nil || tt.err != nil {
			assert.Equal(tt.err.Error(), err.Error())
		}
		assert.Equal(tt.result, c.BodyBuffer)
		assert.Equal(tt.statusCode, c.StatusCode)
		assert.Equal(tt.contentType, c.GetHeader(elton.HeaderContentType))
	}
}
