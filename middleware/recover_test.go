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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestRecoverResponseText(t *testing.T) {
	// panic响应返回text
	assert := assert.New(t)
	var ctx *elton.Context
	e := elton.New()
	e.Use(NewRecover())
	e.GET("/", func(c *elton.Context) error {
		ctx = c
		panic("abc")
	})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	resp := httptest.NewRecorder()
	keys := []string{
		elton.HeaderETag,
		elton.HeaderLastModified,
		elton.HeaderContentEncoding,
		elton.HeaderContentLength,
	}
	for _, key := range keys {
		resp.Header().Set(key, "a")
	}

	catchError := false
	e.OnError(func(_ *elton.Context, _ error) {
		catchError = true
	})

	e.ServeHTTP(resp, req)
	assert.Equal(http.StatusInternalServerError, resp.Code)
	assert.Equal("category=elton-recover, message=abc", resp.Body.String())
	assert.True(ctx.Committed)
	assert.True(catchError)
	for _, key := range keys {
		assert.Empty(ctx.GetHeader(key), "header should be resetted")
	}
}
func TestRecoverResponseJSON(t *testing.T) {
	// 响应返回json
	assert := assert.New(t)
	e := elton.New()
	e.Use(NewRecover())
	e.GET("/", func(c *elton.Context) error {
		panic("abc")
	})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)
	assert.Equal(500, resp.Code)
	assert.Equal(elton.MIMEApplicationJSON, resp.Header().Get(elton.HeaderContentType))
	assert.NotEmpty(resp.Body.Bytes())
}
