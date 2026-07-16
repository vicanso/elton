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

package elton

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteParams(t *testing.T) {
	assert := assert.New(t)
	params := new(RouteParams)
	params.Add("id", "1")
	assert.Equal("1", params.Get("id"))
	assert.Equal(map[string]string{
		"id": "1",
	}, params.ToMap())

	params.Reset()
	assert.Empty(params.Keys)
	assert.Empty(params.Values)
	assert.Equal("", params.Get("id"))
}

func TestNormalizeRoutePath(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("/{$}", normalizeRoutePath(""))
	assert.Equal("/{$}", normalizeRoutePath("/"))
	assert.Equal("/{$}", normalizeRoutePath("/{$}"))
	assert.Equal("/users/{id}", normalizeRoutePath("/users/{id}"))
	assert.Equal("/users/{id}", normalizeRoutePath("/users/:id"))
	assert.Equal("/files/{path...}", normalizeRoutePath("/files/*"))
	assert.Equal("/a/{id}/b/{path...}", normalizeRoutePath("/a/:id/b/*"))
}

func TestExtractParamNames(t *testing.T) {
	assert := assert.New(t)
	assert.Equal([]string{"id"}, extractParamNames("/users/{id}"))
	assert.Equal([]string{"id", "path"}, extractParamNames("/users/{id}/files/{path...}"))
	assert.Empty(extractParamNames("/{$}"))
	assert.Empty(extractParamNames("/static/index.html"))
}
