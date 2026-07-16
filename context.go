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
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vicanso/hes"
)

type (
	// Context elton context
	Context struct {
		Request  *http.Request
		Response http.ResponseWriter
		// Committed commit the data to response, when it's true, the response has been sent.
		// If using custom response handler, please set it true.
		Committed bool
		// ID context id, using unique string function to generate it.
		ID string
		// Route route path, it's equal to the http router path with params.
		Route string
		// Next advances the middleware/handler chain. Set by the framework (stable boundNext);
		// Compose may temporarily replace it. Prefer calling Next() rather than reassigning.
		Next func() error
		// Params route params
		Params *RouteParams
		// StatusCode http response's status code, default is 0 which will be handle as 200
		StatusCode int
		// Body http response's body, which should be converted to bytes by responder middleware.
		// JSON response middleware,  xml response middleware and so on.
		// 约定：io.Reader类型的Body由框架流式写出（实现io.Closer会被自动关闭）；
		// 其它类型需由responder等中间件转换为BodyBuffer后输出
		Body any
		// BodyBuffer http response's body buffer, it should be set by responder middleware.
		BodyBuffer *bytes.Buffer
		// RequestBody http request body, which should be converted by request body parser middleware.
		RequestBody []byte
		// store for context
		m map[any]any
		// realIP the real ip
		realIP string
		// clientIP the clint ip
		clientIP string
		// elton instance
		elton *Elton
		// reuseDisabled whether reuse of the context is disabled
		reuseDisabled atomic.Bool
		// cacheQuery the cache query
		cacheQuery url.Values

		// --- request-scoped chain state (set by framework; not for app use) ---
		// boundNext is allocated once per Context and always points at chainNext.
		boundNext func() error
		// handlers is the immutable snapshot: global middleware + route handlers.
		handlers []Handler
		// handlerIndex is the cursor for Next; starts at -1.
		handlerIndex int
		// handlerNames is used when EnableTrace; reused across requests.
		handlerNames []string
		// activeTrace is non-nil only for the current traced request.
		activeTrace *Trace
	}
)

var _ http.ResponseWriter = (*Context)(nil)

var (
	ErrSignKeyIsNil = hes.New("keys for sign cookie can't be nil")
)

const (
	// SignedCookieSuffix signed cookie suffix
	SignedCookieSuffix = ".sig"
)

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.Context().Deadline()
}

func (c *Context) Done() <-chan struct{} {
	return c.Context().Done()
}

func (c *Context) Err() error {
	return c.Context().Err()
}

func (c *Context) Value(key any) any {
	return c.Context().Value(key)
}

// initBoundNext ensures Next has a stable function that advances the handler chain
// without allocating a new closure on every request.
func (c *Context) initBoundNext() {
	if c.boundNext != nil {
		return
	}
	c.boundNext = func() error { return c.chainNext() }
	c.Next = c.boundNext
}

// chainNext implements the onion-model step for the pre-snapshotted handlers slice.
func (c *Context) chainNext() error {
	if c.Committed {
		return nil
	}
	c.handlerIndex++
	if c.handlerIndex >= len(c.handlers) {
		return nil
	}
	fn := c.handlers[c.handlerIndex]
	if c.activeTrace == nil {
		return fn(c)
	}
	name := ""
	if c.handlerIndex < len(c.handlerNames) {
		name = c.handlerNames[c.handlerIndex]
	}
	if name == "-" {
		return fn(c)
	}
	startedAt := time.Now()
	info := &TraceInfo{
		Name:       name,
		Middleware: true,
	}
	c.activeTrace.Add(info)
	err := fn(c)
	info.Duration = time.Since(startedAt)
	return err
}

// Reset all fields of context
func (c *Context) Reset() {
	c.Request = nil
	c.Response = nil
	c.Committed = false
	c.ID = ""
	c.Route = ""
	c.initBoundNext()
	c.Next = c.boundNext
	c.handlers = nil
	c.handlerIndex = -1
	if c.handlerNames != nil {
		c.handlerNames = c.handlerNames[:0]
	}
	c.activeTrace = nil
	c.Params.Reset()
	c.StatusCode = 0
	c.Body = nil
	c.BodyBuffer = nil
	c.RequestBody = nil
	// Reuse store map to avoid per-request make when handlers use Set/Get.
	if c.m != nil {
		clear(c.m)
	}
	c.realIP = ""
	c.clientIP = ""
	c.reuseDisabled.Store(false)
	c.cacheQuery = nil
}

// GetRemoteAddr returns the remote addr of request
func GetRemoteAddr(req *http.Request) string {
	remoteAddr, _, _ := net.SplitHostPort(req.RemoteAddr)
	return remoteAddr
}

// RemoteAddr returns the remote addr of request
func (c *Context) RemoteAddr() string {
	return GetRemoteAddr(c.Request)
}

// firstCSVField returns the first comma-separated field, trimmed, without allocating a slice.
func firstCSVField(s string) string {
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// lastPublicCSVField returns the rightmost non-intranet hop in a comma-separated list
// (X-Forwarded-For). If every hop is private, returns the first field. No slice alloc.
func lastPublicCSVField(s string) string {
	first, lastPublic := "", ""
	rest := s
	for rest != "" {
		var part string
		if i := strings.IndexByte(rest, ','); i >= 0 {
			part, rest = rest[:i], rest[i+1:]
		} else {
			part, rest = rest, ""
		}
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		if first == "" {
			first = v
		}
		if !IsIntranet(v) {
			lastPublic = v
		}
	}
	if lastPublic != "" {
		return lastPublic
	}
	return first
}

// GetRealIP returns the real ip of request,
// it will get ip from x-forwarded-for from request header,
// if not exists then it will get ip from x-real-ip from request header,
// if not exists then it will use remote addr.
func GetRealIP(req *http.Request) string {
	h := req.Header
	ip := h.Get(HeaderXForwardedFor)
	if ip != "" {
		return firstCSVField(ip)
	}
	ip = h.Get(HeaderXRealIP)
	if ip != "" {
		return ip
	}
	return GetRemoteAddr(req)
}

// RealIP returns the real ip of request,
// it will get ip from x-forwarded-for from request header,
// if not exists then it will get ip from x-real-ip from request header,
// if not exists then it will use remote addr.
func (c *Context) RealIP() string {
	if c.realIP == "" {
		c.realIP = GetRealIP(c.Request)
	}
	return c.realIP
}

// GetClientIP returns the client ip of request,
// it will get ip from x-forwarded-for from request header and get the first public ip,
// if not exists then it will get ip from x-real-ip from request header,
// if not exists then it will use remote addr.
func GetClientIP(req *http.Request) string {
	h := req.Header
	ip := h.Get(HeaderXForwardedFor)
	if ip != "" {
		if v := lastPublicCSVField(ip); v != "" {
			return v
		}
	}
	// x-real-ip为前置设置，如果有，则直接认为是客户IP
	ip = h.Get(HeaderXRealIP)
	if ip != "" {
		return ip
	}
	return GetRemoteAddr(req)
}

// ClientIP returns the client ip of request,
// it will get ip from x-forwared-for from request header and get the first public ip,
// if not exists then it will get ip from x-real-ip from request header,
// if not exists then it will use remote addr.
func (c *Context) ClientIP() string {
	if c.clientIP == "" {
		c.clientIP = GetClientIP(c.Request)
	}
	return c.clientIP
}

// Param returns the route param value.
// Prefer Request.PathValue (ServeMux); fall back to Params for manual/test setups.
func (c *Context) Param(name string) string {
	if c.Request != nil {
		if v := c.Request.PathValue(name); v != "" {
			return v
		}
	}
	if c.Params != nil {
		return c.Params.Get(name)
	}
	return ""
}

// getCacheQuery returns the cache of query
func (c *Context) getCacheQuery() url.Values {
	if c.cacheQuery == nil {
		c.cacheQuery = c.Request.URL.Query()
	}
	return c.cacheQuery
}

// QueryParam returns the query param value
func (c *Context) QueryParam(name string) string {
	query := c.getCacheQuery()
	values := query[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// Query returns the query map.
// It will return map[string]string, not the same as url.Values
// If want to get url.Values, use c.Request.URL.Query()
func (c *Context) Query() map[string]string {
	query := c.getCacheQuery()
	m := make(map[string]string, len(query))
	for key, values := range query {
		m[key] = values[0]
	}
	return m
}

// Redirect the http request to new location
func (c *Context) Redirect(code int, url string) error {
	if code < MinRedirectCode || code > MaxRedirectCode {
		return ErrInvalidRedirect
	}

	c.StatusCode = code
	c.Committed = true
	c.Body = nil
	c.BodyBuffer = nil
	http.Redirect(c.Response, c.Request, url, code)
	return nil
}

// Set the value to the context
func (c *Context) Set(key, value any) {
	if c.m == nil {
		c.m = make(map[any]any, 5)
	}
	c.m[key] = value
}

// Get the value from context
func (c *Context) Get(key any) (any, bool) {
	if c.m == nil {
		return nil, false
	}
	value, exists := c.m[key]
	return value, exists
}

// GetContextValue returns the value of the key from the context store.
// The zero value of T will be returned if the key does not exist
// or the value doesn't match type T.
func GetContextValue[T any](c *Context, key any) T {
	var zero T
	value, exists := c.Get(key)
	if !exists || value == nil {
		return zero
	}
	v, ok := value.(T)
	if !ok {
		return zero
	}
	return v
}

// GetRequestHeader returns header value from http request
func (c *Context) GetRequestHeader(key string) string {
	return c.Request.Header.Get(key)
}

// SetRequestHeader sets http header to request.
// It replaces any existing values of the key.
func (c *Context) SetRequestHeader(key, value string) {
	h := c.Request.Header
	if value == "" {
		h.Del(key)
		return
	}
	h.Set(key, value)
}

// Context returns context of request
func (c *Context) Context() context.Context {
	return c.Request.Context()
}

// WithContext changes the request to new request with context
func (c *Context) WithContext(ctx context.Context) *Context {
	c.Request = c.Request.WithContext(ctx)
	return c
}

// AddRequestHeader adds the key/value to http header.
// It appends to any existing value of the key.
func (c *Context) AddRequestHeader(key, value string) {
	c.Request.Header.Add(key, value)
}

// Header returns headers of http response
func (c *Context) Header() http.Header {
	return c.Response.Header()
}

// WriteHeader sets the http status code
func (c *Context) WriteHeader(statusCode int) {
	c.StatusCode = statusCode
}

// Write the response body
func (c *Context) Write(buf []byte) (int, error) {
	if c.BodyBuffer == nil {
		c.BodyBuffer = new(bytes.Buffer)
	}
	return c.BodyBuffer.Write(buf)
}

// GetHeader returns header value from http response.
// Note: unqualified header methods (GetHeader/SetHeader/AddHeader) operate
// on the RESPONSE header, use GetRequestHeader to read the request header
func (c *Context) GetHeader(key string) string {
	return c.Header().Get(key)
}

// SetHeader sets the key/value to response header.
// It replaces any existing values of the key.
func (c *Context) SetHeader(key, value string) {
	if value == "" {
		c.Header().Del(key)
		return
	}
	c.Header().Set(key, value)
}

// AddHeader adds the key/value to response header.
// It appends to any existing value of the key.
func (c *Context) AddHeader(key, value string) {
	c.Header().Add(key, value)
}

// MergeHeader merges http header to response header
func (c *Context) MergeHeader(h http.Header) {
	for key, values := range h {
		for _, value := range values {
			c.AddHeader(key, value)
		}
	}
}

// ResetHeader resets response header
func (c *Context) ResetHeader() {
	clear(c.Header())
}

// Cookie return the cookie from http request
func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

// AddCookie adds the cookie to the response
func (c *Context) AddCookie(cookie *http.Cookie) {
	http.SetCookie(c, cookie)
}

// SignedCookieWithIndex returns signed cookie from http request
func (c *Context) SignedCookieWithIndex(name string) (*http.Cookie, int, error) {
	cookie, err := c.Cookie(name)
	if err != nil {
		return nil, -1, err
	}
	if c.elton == nil {
		return nil, -1, ErrSignKeyIsNil
	}
	kg := c.elton.keygrip()
	if kg == nil {
		return nil, -1, ErrSignKeyIsNil
	}

	sc, err := c.Cookie(name + SignedCookieSuffix)
	// 如果获取失败，则获取不到cookie
	if err != nil {
		return nil, -1, err
	}
	index := kg.Index([]byte(cookie.Value), []byte(sc.Value))
	return cookie, index, nil
}

// SignedCookie returns signed cookie from http request
func (c *Context) SignedCookie(name string) (*http.Cookie, error) {
	cookie, index, err := c.SignedCookieWithIndex(name)
	if err != nil {
		return cookie, err
	}
	// 如果校验失败，返回无cookie的错误
	if index < 0 {
		cookie = nil
		err = http.ErrNoCookie
	}
	return cookie, err
}

// SendFile to http response
func (c *Context) SendFile(file string) error {
	info, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return err
	}
	if info != nil {
		c.SetHeader(HeaderContentLength, strconv.Itoa(int(info.Size())))
		if c.GetHeader(HeaderLastModified) == "" {
			lmd := info.ModTime().UTC().Format(time.RFC1123)
			c.SetHeader(HeaderLastModified, lmd)
		}
	}
	// elton对于实现了closer的会自动调用关闭
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	c.SetContentTypeByExt(file)
	c.Body = r
	return nil
}

func cloneCookie(cookie *http.Cookie) *http.Cookie {
	return &http.Cookie{
		Name:       cookie.Name,
		Value:      cookie.Value,
		Path:       cookie.Path,
		Domain:     cookie.Domain,
		Expires:    cookie.Expires,
		RawExpires: cookie.RawExpires,
		MaxAge:     cookie.MaxAge,
		Secure:     cookie.Secure,
		HttpOnly:   cookie.HttpOnly,
		SameSite:   cookie.SameSite,
		Raw:        cookie.Raw,
		Unparsed:   cookie.Unparsed,
	}
}

func (c *Context) addSigCookie(cookie *http.Cookie) {
	sc := cloneCookie(cookie)
	sc.Name = sc.Name + SignedCookieSuffix
	if c.elton == nil {
		return
	}
	kg := c.elton.keygrip()
	if kg == nil {
		return
	}
	sc.Value = string(kg.Sign([]byte(sc.Value)))
	c.AddCookie(sc)
}

// AddSignedCookie adds cookie to the response, it will also add a signed cookie
func (c *Context) AddSignedCookie(cookie *http.Cookie) {
	c.AddCookie(cookie)
	c.addSigCookie(cookie)
}

// cleanContent clean content
func (c *Context) cleanContent() {
	c.SetHeader(HeaderContentType, "")
	c.SetHeader(HeaderContentLength, "")
	c.SetHeader(HeaderTransferEncoding, "")
	c.SetHeader(HeaderContentEncoding, "")
	c.Body = nil
	c.BodyBuffer = nil
}

// NoContent clean all content and set status to 204
func (c *Context) NoContent() {
	c.cleanContent()
	c.StatusCode = http.StatusNoContent
}

// NotModified clean all content and set status to 304
func (c *Context) NotModified() {
	c.cleanContent()
	c.StatusCode = http.StatusNotModified
}

// NoCache sets `Cache-Control: no-cache` to the http response header
func (c *Context) NoCache() {
	c.SetHeader(HeaderCacheControl, "no-cache")
}

// NoStore sets `Cache-Control: no-store` to the http response header
func (c *Context) NoStore() {
	c.SetHeader(HeaderCacheControl, "no-store")
}

// CacheMaxAge sets `Cache-Control: public, max-age=MaxAge, s-maxage=SMaxAge` to the http response header.
// If sMaxAge is not empty, it will use the first duration as SMaxAge
func (c *Context) CacheMaxAge(age time.Duration, sMaxAge ...time.Duration) {
	var b strings.Builder
	b.Grow(48)
	b.WriteString("public, max-age=")
	b.WriteString(strconv.Itoa(int(age.Seconds())))
	if len(sMaxAge) != 0 {
		b.WriteString(", s-maxage=")
		b.WriteString(strconv.Itoa(int(sMaxAge[0].Seconds())))
	}
	c.SetHeader(HeaderCacheControl, b.String())
}

// PrivateCacheMaxAge sets `Cache-Control: private, max-age=MaxAge` to the response header.
func (c *Context) PrivateCacheMaxAge(age time.Duration) {
	var b strings.Builder
	b.Grow(32)
	b.WriteString("private, max-age=")
	b.WriteString(strconv.Itoa(int(age.Seconds())))
	c.SetHeader(HeaderCacheControl, b.String())
}

// Created sets the body to response and set the status to 201
func (c *Context) Created(body any) {
	c.StatusCode = http.StatusCreated
	c.Body = body
}

// SetContentTypeByExt sets content type by file extname
func (c *Context) SetContentTypeByExt(file string) {
	ext := filepath.Ext(file)
	contentType := mime.TypeByExtension(ext)
	if contentType != "" {
		c.SetHeader(HeaderContentType, contentType)
	}
}

// ReadFormFile reads the multipart form file data from request
func (c *Context) ReadFormFile(key string) ([]byte, *multipart.FileHeader, error) {
	file, header, err := c.Request.FormFile(key)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = file.Close()
	}()
	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, err
	}
	return buf, header, nil
}

// HTML sets content type and response body as html
func (c *Context) HTML(html string) {
	c.SetContentTypeByExt(".html")
	c.BodyBuffer = bytes.NewBufferString(html)
}

// DisableReuse sets the context disable reuse
func (c *Context) DisableReuse() {
	c.reuseDisabled.Store(true)
}

func (c *Context) isReuse() bool {
	return !c.reuseDisabled.Load()
}

// Elton returns the elton instance of context
func (c *Context) Elton() *Elton {
	return c.elton
}

// Pass request to another elton instance and set the context is committed
func (c *Context) Pass(another *Elton) {
	// 设置为已commit，避免当前实例继续处理
	c.Committed = true
	another.ServeHTTP(c.Response, c.Request)
}

// Pipe the reader to the response
func (c *Context) Pipe(r io.Reader) (int64, error) {
	c.Committed = true
	// 如果是 closer，则需要调用close函数
	closer, ok := r.(io.Closer)
	if ok {
		defer func() {
			_ = closer.Close()
		}()
	}
	return io.Copy(c.Response, r)
}

// IsReaderBody judgets whether body is reader
func (c *Context) IsReaderBody() bool {
	if c.Body == nil {
		return false
	}
	_, ok := c.Body.(io.Reader)
	return ok
}

// closeReaderBody closes the reader body if it implements io.Closer.
// 用于reader body不会被pipe输出的路径（出错或已有BodyBuffer），
// 避免文件句柄等资源泄漏。关闭后将Body置空防止重复关闭
func (c *Context) closeReaderBody() {
	if !c.IsReaderBody() {
		return
	}
	if closer, ok := c.Body.(io.Closer); ok {
		_ = closer.Close()
		c.Body = nil
	}
}

// Trace gets trace from context, if context without trace, new trace will be created.
func (c *Context) Trace() *Trace {
	return TraceFromContext(c.Context())
}

// NewTrace returns a new trace and set it to context value
func (c *Context) NewTrace() *Trace {
	trace := NewTrace()
	c.WithContext(context.WithValue(c.Context(), ContextTraceKey, trace))
	return trace
}

// ServerTiming converts trace info to http response server timing
func (c *Context) ServerTiming(traceInfos TraceInfos, prefix string) {
	value := traceInfos.ServerTiming(prefix)
	if value != "" {
		c.SetHeader(HeaderServerTiming, value)
	}
}

// NewContext return a new context
func NewContext(resp http.ResponseWriter, req *http.Request) *Context {
	c := &Context{}
	c.Request = req
	c.Response = resp
	c.Params = new(RouteParams)
	c.handlerIndex = -1
	c.initBoundNext()
	return c
}
