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
	"net/http"

	"github.com/vicanso/hes"
)

type ContextKey string

const ContextTraceKey ContextKey = "contextTrace"

var (
	methods = []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
		http.MethodTrace,
	}

	// ErrInvalidRedirect invalid redirect
	ErrInvalidRedirect = &hes.Error{
		StatusCode: 400,
		Message:    "invalid redirect",
		Category:   ErrCategory,
	}

	// ErrNilResponse nil response
	ErrNilResponse = &hes.Error{
		StatusCode: 500,
		Message:    "nil response",
		Category:   ErrCategory,
	}
	// ErrNotSupportPush not support http push
	ErrNotSupportPush = &hes.Error{
		StatusCode: 500,
		Message:    "not support http push",
		Category:   ErrCategory,
	}
	// ErrFileNotFound file not found
	ErrFileNotFound = &hes.Error{
		StatusCode: 404,
		Message:    "file not found",
		Category:   ErrCategory,
	}
)

const (
	// ErrCategory elton category
	ErrCategory = "elton"
	// HeaderXForwardedFor x-forwarded-for
	HeaderXForwardedFor = "X-Forwarded-For"
	// HeaderXRealIP x-real-ip
	HeaderXRealIP = "X-Real-Ip"
	// HeaderSetCookie Set-Cookie
	HeaderSetCookie = "Set-Cookie"
	// HeaderLocation Location
	HeaderLocation = "Location"
	// HeaderContentType Content-Type
	HeaderContentType = "Content-Type"
	// HeaderAuthorization Authorization
	HeaderAuthorization = "Authorization"
	// HeaderWWWAuthenticate WWW-Authenticate
	HeaderWWWAuthenticate = "WWW-Authenticate"
	// HeaderCacheControl Cache-Control
	HeaderCacheControl = "Cache-Control"
	// HeaderETag ETag
	HeaderETag = "ETag"
	// HeaderLastModified last modified
	HeaderLastModified = "Last-Modified"
	// HeaderContentEncoding content encoding
	HeaderContentEncoding = "Content-Encoding"
	// HeaderContentLength content length
	HeaderContentLength = "Content-Length"
	// HeaderIfModifiedSince if modified since
	HeaderIfModifiedSince = "If-Modified-Since"
	// HeaderIfNoneMatch if none match
	HeaderIfNoneMatch = "If-None-Match"
	// HeaderAcceptEncoding accept encoding
	HeaderAcceptEncoding = "Accept-Encoding"
	// HeaderServerTiming server timing
	HeaderServerTiming = "Server-Timing"
	// HeaderTransferEncoding transfer encoding
	HeaderTransferEncoding = "Transfer-Encoding"

	// MinRedirectCode min redirect code
	MinRedirectCode = 300
	// MaxRedirectCode max redirect code
	MaxRedirectCode = 308

	// MIMETextPlain text plain
	MIMETextPlain = "text/plain; charset=utf-8"
	// MIMEApplicationJSON application json
	MIMEApplicationJSON = "application/json; charset=utf-8"
	// MIMEBinary binary data
	MIMEBinary = "application/octet-stream"

	// Gzip gzip compress
	Gzip = "gzip"
	// Br brotli compress
	Br = "br"
	// Zstd zstd compress
	Zstd = "zstd"
)

var (
	// ServerTimingDur server timing dur
	ServerTimingDur = []byte(";dur=")
	// ServerTimingDesc server timing desc
	ServerTimingDesc = []byte(`;desc="`)
	// ServerTimingEnd server timing end
	ServerTimingEnd = []byte(`"`)
)
