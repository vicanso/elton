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

package elton

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createRequestHeader(modifiedSince, noneMatch, cacheControl string) http.Header {
	req := httptest.NewRequest("GET", "/users/me", nil)
	header := req.Header
	if modifiedSince != "" {
		header.Set(HeaderIfModifiedSince, modifiedSince)
	}

	if noneMatch != "" {
		header.Set(HeaderIfNoneMatch, noneMatch)
	}

	if cacheControl != "" {
		header.Set(HeaderCacheControl, cacheControl)
	}
	return header
}

func createResponseHeader(lastModified, eTag string) http.Header {
	resp := httptest.NewRecorder()
	header := resp.Header()
	if lastModified != "" {
		header.Set(HeaderLastModified, lastModified)
	}

	if eTag != "" {
		header.Set(HeaderETag, eTag)
	}
	return header
}
func TestFresh(t *testing.T) {
	assert := assert.New(t)
	// when a non-conditional GET is performed
	reqHeader := createRequestHeader("", "", "")
	resHeader := createResponseHeader("", "")
	assert.False(Fresh(reqHeader, resHeader))

	// when ETags match
	reqHeader = createRequestHeader("", "\"foo\"", "")

	resHeader = createResponseHeader("", "\"foo\"")
	assert.True(Fresh(reqHeader, resHeader))

	reqHeader = createRequestHeader("", "W/\"foo\"", "")
	resHeader = createResponseHeader("", "\"foo\"")
	assert.True(Fresh(reqHeader, resHeader))

	reqHeader = createRequestHeader("", "\"foo\"", "")
	resHeader = createResponseHeader("", "W/\"foo\"")
	assert.True(Fresh(reqHeader, resHeader))

	// when ETags mismatch
	reqHeader = createRequestHeader("", "\"foo\"", "")
	resHeader = createResponseHeader("", "\"bar\"")
	assert.False(Fresh(reqHeader, resHeader))

	// when at least one matches
	reqHeader = createRequestHeader("", " \"bar\" , \"foo\"", "")
	resHeader = createResponseHeader("", "\"foo\"")
	assert.True(Fresh(reqHeader, resHeader))

	// when eTag is missing
	reqHeader = createRequestHeader("", "\"foo\"", "")
	resHeader = createResponseHeader("", "")
	assert.False(Fresh(reqHeader, resHeader))

	// when ETag is weak
	reqHeader = createRequestHeader("", "W/\"foo\"", "")
	resHeader = createResponseHeader("", "W/\"foo\"")
	assert.True(Fresh(reqHeader, resHeader))

	resHeader = createResponseHeader("", "\"foo\"")
	assert.True(Fresh(reqHeader, resHeader))

	// when ETag is strong
	reqHeader = createRequestHeader("", "\"foo\"", "")
	resHeader = createResponseHeader("", "\"foo\"")
	assert.True(Fresh(reqHeader, resHeader))

	// weak eTag
	resHeader = createResponseHeader("", "W/\"foo\"")
	assert.True(Fresh(reqHeader, resHeader))

	// when * is given
	reqHeader = createRequestHeader("", "*", "")
	resHeader = createResponseHeader("", "\"foo\"")
	assert.True(Fresh(reqHeader, resHeader))

	reqHeader = createRequestHeader("", "*, \"bar\"", "")
	assert.False(Fresh(reqHeader, resHeader))

	// when modified since the date
	reqHeader = createRequestHeader("Sat, 01 Jan 2000 00:00:00 GMT", "", "")
	resHeader = createResponseHeader("Sat, 01 Jan 2000 01:00:00 GMT", "")
	assert.False(Fresh(reqHeader, resHeader))

	// when unmodified since the date
	reqHeader = createRequestHeader("Sat, 01 Jan 2000 01:00:00 GMT", "", "")
	resHeader = createResponseHeader("Sat, 01 Jan 2000 00:00:00 GMT", "")
	assert.True(Fresh(reqHeader, resHeader))

	// when Last-Modified is missing
	reqHeader = createRequestHeader("Sat, 01 Jan 2000 01:00:00 GMT", "", "")
	resHeader = createResponseHeader("", "")
	assert.False(Fresh(reqHeader, resHeader))

	// with invalid If-Modified-Since date
	reqHeader = createRequestHeader("foo", "", "")
	resHeader = createResponseHeader("Sat, 01 Jan 2000 00:00:00 GMT", "")
	assert.False(Fresh(reqHeader, resHeader))

	// with invalid Last-Modified date
	reqHeader = createRequestHeader("Sat, 01 Jan 2000 00:00:00 GMT", "", "")
	resHeader = createResponseHeader("foo", "")
	assert.False(Fresh(reqHeader, resHeader))

	// when requested with If-Modified-Since and If-None-Match

	// both match
	reqHeader = createRequestHeader("Sat, 01 Jan 2000 00:00:00 GMT", "\"foo\"", "")
	resHeader = createResponseHeader("Sat, 01 Jan 2000 00:00:00 GMT", "\"foo\"")
	assert.True(Fresh(reqHeader, resHeader))

	// when only ETag matches
	reqHeader = createRequestHeader("Sat, 01 Jan 2000 00:00:00 GMT", "\"foo\"", "")
	resHeader = createResponseHeader("Sat, 01 Jan 2000 01:00:00 GMT", "\"foo\"")
	assert.False(Fresh(reqHeader, resHeader))

	// when only Last-Modified matches
	reqHeader = createRequestHeader("Sat, 01 Jan 2000 00:00:00 GMT", "\"foo\"", "")
	resHeader = createResponseHeader("Sat, 01 Jan 2000 00:00:00 GMT", "\"bar\"")
	assert.False(Fresh(reqHeader, resHeader))

	// when none match
	reqHeader = createRequestHeader("Sat, 01 Jan 2000 00:00:00 GMT", "\"foo\"", "")
	resHeader = createResponseHeader("Sat, 01 Jan 2000 01:00:00 GMT", "\"bar\"")
	assert.False(Fresh(reqHeader, resHeader))

	// when requested with Cache-Control: no-cache
	reqHeader = createRequestHeader("", "", "no-cache")
	resHeader = createResponseHeader("", "")
	assert.False(Fresh(reqHeader, resHeader))

	// when ETags match
	reqHeader = createRequestHeader("", "\"foo\"", "no-cache")
	resHeader = createResponseHeader("", "\"foo\"")
	assert.False(Fresh(reqHeader, resHeader))

	// when unmodified since the date
	reqHeader = createRequestHeader("Sat, 01 Jan 2000 00:00:00 GMT", "", "no-cache")
	resHeader = createResponseHeader("Sat, 01 Jan 2000 00:00:00 GMT", "\"foo\"")
	assert.False(Fresh(reqHeader, resHeader))
}

func BenchmarkFresh(b *testing.B) {
	b.ResetTimer()
	reqHeader := createRequestHeader("Sat, 01 Jan 2000 00:00:00 GMT", "\"foo\"", "")
	resHeader := createResponseHeader("Sat, 01 Jan 2000 00:00:00 GMT", "\"foo\"")
	for i := 0; i < b.N; i++ {
		Fresh(reqHeader, resHeader)
	}
}
