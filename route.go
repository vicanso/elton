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
	"bufio"
	"net"
	"net/http"
	"strings"
	"sync"
)

// normalizeRoutePath converts limited legacy chi-style segments to net/http ServeMux patterns.
//
//   - ":name" at a path segment start → "{name}"
//   - trailing "/*" → "/{path...}" (catch-all; read with Param("path") or Params.Values[0])
//
// Patterns already using ServeMux syntax ({name}, {name...}, {$}) are left unchanged.
// Regexp constraints like {id:[0-9]+} are not supported and will panic at registration.
func normalizeRoutePath(path string) string {
	if path == "" || path == "/" {
		// Bare "/" is a prefix catch-all in ServeMux; elton treats it as exact root.
		return "/{$}"
	}
	if base, ok := strings.CutSuffix(path, "/*"); ok {
		path = base + "/{path...}"
	}
	if !strings.Contains(path, ":") {
		return path
	}
	var b strings.Builder
	b.Grow(len(path) + 4)
	for i := 0; i < len(path); {
		// Convert :name at segment start (after '/' or at beginning).
		if path[i] == ':' && (i == 0 || path[i-1] == '/') {
			j := i + 1
			for j < len(path) && isRouteIdentByte(path[j]) {
				j++
			}
			if j > i+1 {
				b.WriteByte('{')
				b.WriteString(path[i+1 : j])
				b.WriteByte('}')
				i = j
				continue
			}
		}
		b.WriteByte(path[i])
		i++
	}
	return b.String()
}

// extractParamNames returns wildcard names from a ServeMux path pattern, in order.
// {$} and anonymous trailing-slash multis are omitted.
func extractParamNames(path string) []string {
	var names []string
	for i := 0; i < len(path); i++ {
		if path[i] != '{' {
			continue
		}
		end := strings.IndexByte(path[i:], '}')
		if end < 0 {
			break
		}
		name := path[i+1 : i+end]
		i += end
		if name == "" || name == "$" {
			continue
		}
		if n, ok := strings.CutSuffix(name, "..."); ok {
			name = n
		}
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func isRouteIdentByte(c byte) bool {
	return c == '_' ||
		(c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9')
}

// edgeWriter wraps the real ResponseWriter so ServeMux is invoked once per request.
//
// Elton-registered handlers call markHandled() before writing. Mux-internal handlers
// (404 / 405 / trailing-slash redirect) leave handled=false; their headers/body stay
// buffered until ServeHTTP decides whether to flush or replace with elton's handlers.
type edgeWriter struct {
	http.ResponseWriter
	hdr         http.Header // delayed headers while !handled
	status      int
	handled     bool
	headerWrote bool
	buf         []byte
}

var edgeWriterPool = sync.Pool{
	New: func() any {
		return &edgeWriter{}
	},
}

func acquireEdgeWriter(w http.ResponseWriter) *edgeWriter {
	ew := edgeWriterPool.Get().(*edgeWriter)
	ew.ResponseWriter = w
	ew.hdr = nil
	ew.status = 0
	ew.handled = false
	ew.headerWrote = false
	ew.buf = ew.buf[:0]
	return ew
}

func releaseEdgeWriter(ew *edgeWriter) {
	if ew == nil {
		return
	}
	ew.ResponseWriter = nil
	ew.hdr = nil
	ew.buf = ew.buf[:0]
	edgeWriterPool.Put(ew)
}

// detachEdgeFromContext peels *edgeWriter off c.Response so the writer can be
// returned to the pool while Context (tests / DisableReuse) still has a usable Header().
func detachEdgeFromContext(c *Context) {
	if c == nil {
		return
	}
	if ew, ok := c.Response.(*edgeWriter); ok {
		c.Response = ew.ResponseWriter
	}
}

func (ew *edgeWriter) Header() http.Header {
	if ew.handled {
		return ew.ResponseWriter.Header()
	}
	if ew.hdr == nil {
		ew.hdr = make(http.Header)
	}
	return ew.hdr
}

func (ew *edgeWriter) markHandled() {
	if ew.handled {
		return
	}
	ew.handled = true
	if ew.hdr != nil {
		dst := ew.ResponseWriter.Header()
		for k, vv := range ew.hdr {
			dst[k] = vv
		}
		ew.hdr = nil
	}
	// Route handlers should not have written before markHandled.
	ew.buf = ew.buf[:0]
	ew.status = 0
	ew.headerWrote = false
}

func (ew *edgeWriter) WriteHeader(statusCode int) {
	if ew.headerWrote {
		return
	}
	ew.status = statusCode
	if ew.handled {
		ew.headerWrote = true
		ew.ResponseWriter.WriteHeader(statusCode)
	}
}

func (ew *edgeWriter) Write(b []byte) (int, error) {
	if ew.handled {
		if !ew.headerWrote {
			if ew.status == 0 {
				ew.status = http.StatusOK
			}
			ew.headerWrote = true
			ew.ResponseWriter.WriteHeader(ew.status)
		}
		return ew.ResponseWriter.Write(b)
	}
	if ew.status == 0 {
		ew.status = http.StatusOK
	}
	ew.buf = append(ew.buf, b...)
	return len(b), nil
}

// flush writes buffered mux-internal response (e.g. trailing-slash redirect).
func (ew *edgeWriter) flush() {
	if ew.hdr != nil {
		dst := ew.ResponseWriter.Header()
		for k, vv := range ew.hdr {
			dst[k] = vv
		}
		ew.hdr = nil
	}
	if !ew.headerWrote {
		code := ew.status
		if code == 0 {
			code = http.StatusOK
		}
		ew.headerWrote = true
		ew.ResponseWriter.WriteHeader(code)
	}
	if len(ew.buf) > 0 {
		_, _ = ew.ResponseWriter.Write(ew.buf)
	}
}

// Unwrap supports http.ResponseController and middleware that peel wrappers.
func (ew *edgeWriter) Unwrap() http.ResponseWriter {
	return ew.ResponseWriter
}

func (ew *edgeWriter) Flush() {
	if f, ok := ew.ResponseWriter.(http.Flusher); ok {
		if ew.handled && !ew.headerWrote {
			ew.WriteHeader(http.StatusOK)
		}
		f.Flush()
	}
}

func (ew *edgeWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := ew.ResponseWriter.(http.Hijacker); ok {
		ew.markHandled()
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// markRouteHandled marks the ResponseWriter chain as an elton-registered route.
// Walks Unwrap() so Server/test wrappers do not hide *edgeWriter.
func markRouteHandled(w http.ResponseWriter) {
	for w != nil {
		if ew, ok := w.(*edgeWriter); ok {
			ew.markHandled()
			return
		}
		uw, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return
		}
		w = uw.Unwrap()
	}
}
