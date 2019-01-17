// Copyright 2018 tree xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cod

import (
	"net/http"

	"github.com/vicanso/hes"
)

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
		Category:   ErrCategoryCod,
	}
	// ErrInvalidResponse invalid response(body an status is nil)
	ErrInvalidResponse = &hes.Error{
		StatusCode: 500,
		Message:    "invalid response",
		Category:   ErrCategoryCod,
	}
	// ErrNillResponse nil response
	ErrNillResponse = &hes.Error{
		StatusCode: 500,
		Message:    "nil response",
		Category:   ErrCategoryCod,
	}
)

const (
	// ErrCategoryCod cod category
	ErrCategoryCod = "cod"
	// HeaderXForwardedFor x-forwarded-for
	HeaderXForwardedFor = "X-Forwarded-For"
	// HeaderXRealIp x-real-ip
	HeaderXRealIp = "X-Real-Ip"
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
	// HeaderIfModifiedSince if modified since
	HeaderIfModifiedSince = "If-Modified-Since"
	// HeaderIfNoneMatch if none match
	HeaderIfNoneMatch = "If-None-Match"
	// HeaderAcceptEncoding accept encoding
	HeaderAcceptEncoding = "Accept-Encoding"
	// HeaderServerTiming server timing
	HeaderServerTiming = "Server-Timing"

	// MinRedirectCode min redirect code
	MinRedirectCode = 300
	// MaxRedirectCode max redirect code
	MaxRedirectCode = 308

	// MIMETextPlain text plain
	MIMETextPlain = "text/plain;charset=UTF-8"
	// MIMEApplicationJSON application json
	MIMEApplicationJSON = "application/json;charset=UTF-8"
	// MIMEBinary binary data
	MIMEBinary = "application/octet-stream"

	// Gzip gzip compress
	Gzip = "gzip"
	// Br brotli compress
	Br = "br"
)

var (
	// ServerTimingDur server timing dur
	ServerTimingDur = []byte(";dur=")
	// ServerTimingDesc server timing desc
	ServerTimingDesc = []byte(`;desc="`)
	// ServerTimingEnd server timing end
	ServerTimingEnd = []byte(`"`)
)
