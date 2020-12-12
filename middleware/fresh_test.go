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
	assert := assert.New(t)
	skipErr := errors.New("skip error")
	// next直接返回skip error，用于判断是否执行了next
	next := func() error {
		return skipErr
	}
	defaultFresh := NewDefaultFresh()
	tests := []struct {
		newContext func() *elton.Context
		err        error
		statusCode int
		body       interface{}
		result     *bytes.Buffer
	}{
		// skip
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(nil, nil)
				c.Committed = true
				c.Next = next
				return c

			},
			err: skipErr,
		},
		// error
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(nil, nil)
				c.Next = next
				return c

			},
			err: skipErr,
		},
		// pass method
		{
			newContext: func() *elton.Context {
				modifiedAt := "Tue, 25 Dec 2018 00:02:22 GMT"

				req := httptest.NewRequest("POST", "/users/me", nil)
				req.Header.Set(elton.HeaderIfModifiedSince, modifiedAt)
				resp := httptest.NewRecorder()
				resp.Header().Set(elton.HeaderLastModified, modifiedAt)

				c := elton.NewContext(resp, req)
				c.Next = func() error {
					c.StatusCode = http.StatusOK
					c.Body = map[string]string{
						"name": "tree.xie",
					}
					c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
					return nil
				}
				return c
			},
			statusCode: 200,
			body: map[string]string{
				"name": "tree.xie",
			},
			result: bytes.NewBufferString(`{"name":"tree.xie"}`),
		},
		// status code >= 300
		{
			newContext: func() *elton.Context {
				modifiedAt := "Tue, 25 Dec 2018 00:02:22 GMT"

				req := httptest.NewRequest("GET", "/users/me", nil)
				req.Header.Set(elton.HeaderIfModifiedSince, modifiedAt)
				resp := httptest.NewRecorder()
				resp.Header().Set(elton.HeaderLastModified, modifiedAt)

				c := elton.NewContext(resp, req)
				c.Next = func() error {
					c.StatusCode = http.StatusBadRequest
					c.Body = map[string]string{
						"name": "tree.xie",
					}
					c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
					return nil
				}
				return c
			},
			statusCode: http.StatusBadRequest,
			body: map[string]string{
				"name": "tree.xie",
			},
			result: bytes.NewBufferString(`{"name":"tree.xie"}`),
		},
		// 304
		{
			newContext: func() *elton.Context {
				modifiedAt := "Tue, 25 Dec 2018 00:02:22 GMT"

				req := httptest.NewRequest("GET", "/users/me", nil)
				req.Header.Set(elton.HeaderIfModifiedSince, modifiedAt)
				resp := httptest.NewRecorder()
				resp.Header().Set(elton.HeaderLastModified, modifiedAt)

				c := elton.NewContext(resp, req)
				c.Next = func() error {
					c.Body = map[string]string{
						"name": "tree.xie",
					}
					c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
					return nil
				}
				return c
			},
			statusCode: 304,
		},
	}

	for _, tt := range tests {
		c := tt.newContext()
		err := defaultFresh(c)
		assert.Equal(tt.err, err)
		assert.Equal(tt.result, c.BodyBuffer)
		assert.Equal(tt.body, c.Body)
	}
}
