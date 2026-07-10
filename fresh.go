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
	"regexp"
	"strings"
	"time"
)

var noCacheReg = regexp.MustCompile(`(?:^|,)\s*?no-cache\s*?(?:,|$)`)

const weakTagPrefix = "W/"

func parseHTTPDate(date string) int64 {
	t, err := time.Parse(time.RFC1123, date)
	if err != nil {
		return 0
	}
	return t.Unix()
}

// etagWeakMatch 弱ETag比较：W/"xxx" 与 "xxx" 任意一侧带W/前缀均视为匹配
// （弱比较语义，见RFC 7232 2.3.2）
func etagWeakMatch(match, etag string) bool {
	if match == etag {
		return true
	}
	if strings.HasPrefix(match, weakTagPrefix) && match[2:] == etag {
		return true
	}
	if strings.HasPrefix(etag, weakTagPrefix) && etag[2:] == match {
		return true
	}
	return false
}

// isFresh returns true if the data is fresh
func isFresh(modifiedSince, noneMatch, cacheControl, lastModified, etag string) bool {
	if modifiedSince == "" && noneMatch == "" {
		return false
	}
	// 请求端指定no-cache时不使用缓存
	if cacheControl != "" && noCacheReg.MatchString(cacheControl) {
		return false
	}
	// if none match
	// "*" 表示匹配任意etag，跳过比较
	if noneMatch != "" && noneMatch != "*" {
		if etag == "" {
			return false
		}
		// If-None-Match可为逗号分隔的多个etag，任意一个弱匹配即为fresh
		etagStale := true
		for match := range strings.SplitSeq(noneMatch, ",") {
			if etagWeakMatch(strings.TrimSpace(match), etag) {
				etagStale = false
				break
			}
		}
		if etagStale {
			return false
		}
	}
	// if modified since
	if modifiedSince != "" {
		if lastModified == "" {
			return false
		}
		lastModifiedUnix := parseHTTPDate(lastModified)
		modifiedSinceUnix := parseHTTPDate(modifiedSince)
		if lastModifiedUnix == 0 || modifiedSinceUnix == 0 {
			return false
		}
		if modifiedSinceUnix < lastModifiedUnix {
			return false
		}
	}
	return true
}

// Fresh returns fresh status by judging request header and response header
func Fresh(reqHeader http.Header, resHeader http.Header) bool {
	return isFresh(
		reqHeader.Get(HeaderIfModifiedSince),
		reqHeader.Get(HeaderIfNoneMatch),
		reqHeader.Get(HeaderCacheControl),
		resHeader.Get(HeaderLastModified),
		resHeader.Get(HeaderETag),
	)
}
