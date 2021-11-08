// MIT License

// Copyright (c) 2021 Tree Xie

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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheResponse(t *testing.T) {
	assert := assert.New(t)

	cp := &CacheResponse{
		StatusCode: 200,
		Header: http.Header{
			"Cache-Control": []string{
				"no-cache",
			},
			"Content-Type": []string{
				"application/json",
			},
		},
		Body: bytes.NewBufferString("abcd"),
	}
	data := cp.Bytes()
	assert.Equal(37, len(data))

	cp = NewCacheResponse(data)
	assert.Equal(200, cp.StatusCode)
	assert.Equal(http.Header{
		"Cache-Control": []string{
			"no-cache",
		},
		"Content-Type": []string{
			"application/json",
		},
	}, cp.Header)
	assert.Equal("abcd", cp.Body.String())
}
