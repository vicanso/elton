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
	"bytes"
	"net/http"
	"regexp"
	"time"
)

var noCacheReg = regexp.MustCompile(`(?:^|,)\s*?no-cache\s*?(?:,|$)`)

var weekTagPrefix = []byte("W/")

func parseTokenList(buf []byte) [][]byte {
	end := 0
	start := 0
	count := len(buf)
	list := make([][]byte, 0)
	for index := 0; index < count; index++ {
		switch int(buf[index]) {
		// 空格
		case 0x20:
			if start == end {
				end = index + 1
				start = end
			}
		// , 号
		case 0x2c:
			list = append(list, buf[start:end])
			end = index + 1
			start = end
		default:
			end = index + 1
		}
	}
	list = append(list, buf[start:end])
	return list
}

func parseHTTPDate(date string) int64 {
	t, err := time.Parse(time.RFC1123, date)
	if err != nil {
		return 0
	}
	return t.Unix()
}

// isFresh returns true if the data is fresh
func isFresh(modifiedSince, noneMatch, cacheControl, lastModified, etag []byte) bool {
	if len(modifiedSince) == 0 && len(noneMatch) == 0 {
		return false
	}
	if len(cacheControl) != 0 && noCacheReg.Match(cacheControl) {
		return false
	}
	// if none match
	if len(noneMatch) != 0 && (len(noneMatch) != 1 || noneMatch[0] != byte('*')) {
		if len(etag) == 0 {
			return false
		}
		matches := parseTokenList(noneMatch)
		etagStale := true
		for _, match := range matches {
			if bytes.Equal(match, etag) {
				etagStale = false
				break
			}
			if bytes.HasPrefix(match, weekTagPrefix) && bytes.Equal(match[2:], etag) {
				etagStale = false
				break
			}
			if bytes.HasPrefix(etag, weekTagPrefix) && bytes.Equal(etag[2:], match) {
				etagStale = false
				break
			}
		}
		if etagStale {
			return false
		}
	}
	// if modified since
	if len(modifiedSince) != 0 {
		if len(lastModified) == 0 {
			return false
		}
		lastModifiedUnix := parseHTTPDate(string(lastModified))
		modifiedSinceUnix := parseHTTPDate(string(modifiedSince))
		if lastModifiedUnix == 0 || modifiedSinceUnix == 0 {
			return false
		}
		if modifiedSinceUnix < lastModifiedUnix {
			return false
		}
	}
	return true
}

// Fresh returns fresh status by judget request header and response header
func Fresh(reqHeader http.Header, resHeader http.Header) bool {
	modifiedSince := []byte(reqHeader.Get(HeaderIfModifiedSince))
	noneMatch := []byte(reqHeader.Get(HeaderIfNoneMatch))
	cacheControl := []byte(reqHeader.Get(HeaderCacheControl))

	lastModified := []byte(resHeader.Get(HeaderLastModified))
	etag := []byte(resHeader.Get(HeaderETag))

	return isFresh(modifiedSince, noneMatch, cacheControl, lastModified, etag)
}
