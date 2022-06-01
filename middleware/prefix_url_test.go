// MIT License

// Copyright (c) 2022 Tree Xie

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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPrefixURL(t *testing.T) {
	assert := assert.New(t)

	fn := NewPrefixURL("/api", `/\d{2,5}`)

	tests := []struct {
		url    string
		result string
	}{
		{
			url:    "/api/users/v1/me",
			result: "/users/v1/me",
		},
		{
			url:    "/rest/api/users/v1/me",
			result: "/rest/api/users/v1/me",
		},
		{
			url:    "/100/users/v1/me",
			result: "/users/v1/me",
		},
		{
			url:    "/1/users/v1/me",
			result: "/1/users/v1/me",
		},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.url, nil)
		fn(req)
		assert.Equal(tt.result, req.URL.Path)
	}
}
