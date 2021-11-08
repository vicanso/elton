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
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShortHeaderIndexes(t *testing.T) {
	assert := assert.New(t)

	name := shortHeaderIndexes.getName(1)
	assert.Equal("cache-control", name)
	name = shortHeaderIndexes.getName(3)
	assert.Equal("content-type", name)
	assert.Empty(shortHeaderIndexes.getName(int(MaxShortHeader)))

	// 故意大写了O
	index, ok := shortHeaderIndexes.getIndex("Cache-COntrol")
	assert.True(ok)
	assert.Equal(index, uint8(1))

	_, ok = shortHeaderIndexes.getIndex("abc")
	assert.False(ok)
}

func TestHTTPHeader(t *testing.T) {
	assert := assert.New(t)

	// 压缩的header
	h := NewHTTPHeader("Cache-Control", []string{"no-cache"})
	assert.Equal(uint8(1), h[0])
	assert.Equal("no-cache", string(h[1:]))

	name, values := h.Header()
	assert.Equal("cache-control", name)
	assert.Equal([]string{
		"no-cache",
	}, values)

	// 非压缩的header
	h = NewHTTPHeader("X-Test", []string{"my name", "my job"})
	assert.Equal(uint8(NoneMatchHeader), h[0])
	assert.Equal("X-Test:my name\nmy job", string(h[1:]))
	name, values = h.Header()
	assert.Equal("X-Test", name)
	assert.Equal([]string{
		"my name",
		"my job",
	}, values)
}

func TestHTTPHeaders(t *testing.T) {
	assert := assert.New(t)
	header := http.Header{
		"Cache-Control": []string{
			"max-age=0, private, must-revalidate",
		},
		"Content-Encoding": []string{
			"gzip",
		},
		"Content-Type": []string{
			"text/html; charset=utf-8",
		},
		"Date": []string{
			"Mon, 08 Nov 2021 23:48:55 GMT",
		},
		"Etag": []string{
			`W/"e232d5a706265f21a7019b5ab453e14a"`,
		},
		"X-Referrer-Policy": []string{
			"origin-when-cross-origin, strict-origin-when-cross-origin",
		},
		"X-Trace-Id": []string{
			"83C9:30C1:9C96A:146FC2:61888203",
		},
	}
	hs := NewHTTPHeaders(header)
	assert.Equal(258, len(hs))
	assert.Equal(header, hs.Header())
}

func BenchmarkNewShortHTTPHeader(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewHTTPHeader("Cache-Control", []string{"no-cache"})
	}
}

func BenchmarkNewHTTPHeader(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewHTTPHeader("X-Test", []string{"my name", "my job"})
	}
}

func getTestHTTPHeader() http.Header {
	return http.Header{
		"Cache-Control": []string{
			"max-age=0, private, must-revalidate",
		},
		"Content-Encoding": []string{
			"gzip",
		},
		"Content-Type": []string{
			"text/html; charset=utf-8",
		},
		"Date": []string{
			"Mon, 08 Nov 2021 23:48:55 GMT",
		},
		"Etag": []string{
			`W/"e232d5a706265f21a7019b5ab453e14a"`,
		},
		"X-Referrer-Policy": []string{
			"origin-when-cross-origin, strict-origin-when-cross-origin",
		},
		"X-Trace-Id": []string{
			"83C9:30C1:9C96A:146FC2:61888203",
		},
	}
}

func BenchmarkNewHTTPHeaders(b *testing.B) {
	header := getTestHTTPHeader()

	for i := 0; i < b.N; i++ {
		_ = NewHTTPHeaders(header)
	}
}

func BenchmarkHTTPHeaderMarshal(b *testing.B) {
	header := getTestHTTPHeader()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(header)
	}
}

func BenchmarkToHTTPHeader(b *testing.B) {
	hs := NewHTTPHeaders(getTestHTTPHeader())
	for i := 0; i < b.N; i++ {
		_ = hs.Header()
	}
}

func BenchmarkHTTPHeaderUnmarshal(b *testing.B) {
	buf, _ := json.Marshal(getTestHTTPHeader())
	if len(buf) == 0 {
		panic("marshal fail")
	}
	for i := 0; i < b.N; i++ {
		header := make(http.Header)
		_ = json.Unmarshal(buf, &header)
	}
}
