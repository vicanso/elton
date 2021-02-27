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
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vicanso/hes"
	intranetip "github.com/vicanso/intranet-ip"
	"github.com/vicanso/keygrip"
)

const (
	// ReuseContextEnabled resuse context enabled
	ReuseContextEnabled int32 = iota
	// ReuseContextDisabled reuse context disabled
	ReuseContextDisabled
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
		// Next next function, it will be auto generated.
		Next func() error
		// Params route params
		Params *RouteParams
		// StatusCode http response's status code, default is 0 which will be handle as 200
		StatusCode int
		// Body http response's body, which should be converted to bytes by responder middleware.
		// JSON response middleware,  xml response middleware and so on.
		Body interface{}
		// BodyBuffer http response's body buffer, it should be set by responder middleware.
		BodyBuffer *bytes.Buffer
		// RequestBody http request body, which should be converted by request body parser middleware.
		RequestBody []byte
		// store for context
		m map[interface{}]interface{}
		// realIP the real ip
		realIP string
		// clientIP the clint ip
		clientIP string
		// elton instance
		elton *Elton
		// reuseStatus reuse status
		reuseStatus int32
		// cacheQuery the cache query
		cacheQuery url.Values
	}
)

var _ http.ResponseWriter = (*Context)(nil)

var (
	errSignKeyIsNil = hes.New("keys for sign cookie can't be nil")
)

const (
	// SignedCookieSuffix signed cookie suffix
	SignedCookieSuffix = ".sig"
)

// Reset all fields of context
func (c *Context) Reset() {
	c.Request = nil
	c.Response = nil
	c.Committed = false
	c.ID = ""
	c.Route = ""
	c.Next = nil
	c.Params.Reset()
	c.StatusCode = 0
	c.Body = nil
	c.BodyBuffer = nil
	c.RequestBody = nil
	c.m = nil
	c.realIP = ""
	c.clientIP = ""
	c.reuseStatus = ReuseContextEnabled
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

// GetRealIP returns the real ip of request,
// it will get ip from x-forwared-for from request header,
// if not exists then it will get ip from x-real-ip from request header,
// if not exists then it will use remote addr.
func GetRealIP(req *http.Request) string {
	h := req.Header
	ip := h.Get(HeaderXForwardedFor)
	if ip != "" {
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	ip = h.Get(HeaderXRealIP)
	if ip != "" {
		return ip
	}
	return GetRemoteAddr(req)
}

// RealIP returns the real ip of request,
// it will get ip from x-forwared-for from request header,
// if not exists then it will get ip from x-real-ip from request header,
// if not exists then it will use remote addr.
func (c *Context) RealIP() string {
	if c.realIP != "" {
		return c.realIP
	}
	c.realIP = GetRealIP(c.Request)
	return c.realIP
}

// GetClientIP returns the client ip of request,
// it will get ip from x-forwared-for from request header and get the first public ip,
// if not exists then it will get ip from x-real-ip from request header,
// if not exists then it will use remote addr.
func GetClientIP(req *http.Request) string {
	h := req.Header
	ip := h.Get(HeaderXForwardedFor)
	if ip != "" {
		arr := sort.StringSlice(strings.Split(ip, ","))
		// 从后往前找第一个非内网IP的则为客户IP
		for i := len(arr) - 1; i >= 0; i-- {
			v := strings.TrimSpace(arr[i])
			if !intranetip.Is(net.ParseIP(v)) {
				return v
			}
		}
		// 如果所有IP都是内网IP，则直接取第一个
		if len(arr) != 0 {
			return strings.TrimSpace(arr[0])
		}
	}
	ip = h.Get(HeaderXRealIP)
	if ip != "" {
		if !intranetip.Is(net.ParseIP(ip)) {
			return ip
		}
	}
	return GetRemoteAddr(req)
}

// ClientIP returns the client ip of request,
// it will get ip from x-forwared-for from request header and get the first public ip,
// if not exists then it will get ip from x-real-ip from request header,
// if not exists then it will use remote addr.
func (c *Context) ClientIP() string {
	if c.clientIP != "" {
		return c.clientIP
	}
	c.clientIP = GetClientIP(c.Request)
	return c.clientIP
}

// Param returns the route param value
func (c *Context) Param(name string) string {
	if c.Params == nil {
		return ""
	}
	return c.Params.Get(name)
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
func (c *Context) Redirect(code int, url string) (err error) {
	if code < MinRedirectCode || code > MaxRedirectCode {
		err = ErrInvalidRedirect
		return
	}

	c.StatusCode = code
	c.Committed = true
	c.Body = nil
	c.BodyBuffer = nil
	http.Redirect(c.Response, c.Request, url, code)
	return
}

// Set the value to the context
func (c *Context) Set(key, value interface{}) {
	if c.m == nil {
		c.m = make(map[interface{}]interface{}, 5)
	}
	c.m[key] = value
}

// Get the value from context
func (c *Context) Get(key interface{}) (value interface{}, exists bool) {
	if c.m == nil {
		return nil, false
	}
	value, exists = c.m[key]
	return
}

// GetInt returns int value from context
func (c *Context) GetInt(key interface{}) (i int) {
	if value, exists := c.Get(key); exists && value != nil {
		i, _ = value.(int)
	}
	return
}

// GetInt64 returns int64 value from context
func (c *Context) GetInt64(key interface{}) (i int64) {
	if value, exists := c.Get(key); exists && value != nil {
		i, _ = value.(int64)
	}
	return
}

// GetString returns string value from context
func (c *Context) GetString(key interface{}) (s string) {
	if value, exists := c.Get(key); exists && value != nil {
		s, _ = value.(string)
	}
	return
}

// GetBool returns bool value from context
func (c *Context) GetBool(key interface{}) (b bool) {
	if value, exists := c.Get(key); exists && value != nil {
		b, _ = value.(bool)
	}
	return
}

// GetFloat32 returns float32 value from context
func (c *Context) GetFloat32(key interface{}) (f float32) {
	if value, exists := c.Get(key); exists && value != nil {
		f, _ = value.(float32)
	}
	return
}

// GetFloat64 returns float64 value from context
func (c *Context) GetFloat64(key interface{}) (f float64) {
	if value, exists := c.Get(key); exists && value != nil {
		f, _ = value.(float64)
	}
	return
}

// GetTime returns time value from context
func (c *Context) GetTime(key interface{}) (t time.Time) {
	if value, exists := c.Get(key); exists && value != nil {
		t, _ = value.(time.Time)
	}
	return
}

// GetDuration returns duration from context
func (c *Context) GetDuration(key interface{}) (d time.Duration) {
	if value, exists := c.Get(key); exists && value != nil {
		d, _ = value.(time.Duration)
	}
	return
}

// GetStringSlice returns string slice from context
func (c *Context) GetStringSlice(key interface{}) (arr []string) {
	if value, exists := c.Get(key); exists && value != nil {
		arr, _ = value.([]string)
	}
	return
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

// GetHeader return header value from http response
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
	h := c.Header()
	for k := range h {
		h.Del(k)
	}
}

// Cookie return the cookie from http request
func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

// AddCookie adds the cookie to the response
func (c *Context) AddCookie(cookie *http.Cookie) {
	http.SetCookie(c, cookie)
}

func (c *Context) getKeys() []string {
	e := c.elton
	if e == nil || e.SignedKeys == nil {
		return nil
	}
	return e.SignedKeys.GetKeys()
}

// GetSignedCookie returns signed cookie from http request
func (c *Context) GetSignedCookie(name string) (cookie *http.Cookie, index int, err error) {
	index = -1
	cookie, err = c.Cookie(name)
	if err != nil {
		return
	}
	keys := c.getKeys()
	if len(keys) == 0 {
		err = errSignKeyIsNil
		return
	}

	sc, err := c.Cookie(name + SignedCookieSuffix)
	if err != nil {
		cookie = nil
		return
	}
	kg := keygrip.New(keys)
	index = kg.Index([]byte(cookie.Value), []byte(sc.Value))
	return
}

// SignedCookie returns signed cookie from http request
func (c *Context) SignedCookie(name string) (cookie *http.Cookie, err error) {
	cookie, index, err := c.GetSignedCookie(name)
	if err != nil {
		return
	}
	if index < 0 {
		cookie = nil
		err = http.ErrNoCookie
	}
	return
}

// SendFile to http response
func (c *Context) SendFile(file string) (err error) {
	info, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			err = ErrFileNotFound
		}
		return
	}
	if info != nil {
		c.SetHeader(HeaderContentLength, strconv.Itoa(int(info.Size())))
		if c.GetHeader(HeaderLastModified) == "" {
			lmd := info.ModTime().UTC().Format(time.RFC1123)
			c.SetHeader(HeaderLastModified, lmd)
		}
	}
	r, err := os.Open(file)
	if err != nil {
		return
	}
	c.SetContentTypeByExt(file)
	c.Body = r
	return
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
	keys := c.getKeys()
	if len(keys) == 0 {
		return
	}
	kg := keygrip.New(keys)
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

// NoCache set `Cache-Control: no-cache` to the http response header
func (c *Context) NoCache() {
	c.SetHeader(HeaderCacheControl, "no-cache")
}

// NoStore set `Cache-Control: no-store` to the http response header
func (c *Context) NoStore() {
	c.SetHeader(HeaderCacheControl, "no-store")
}

// CacheMaxAge set `Cache-Control: public, max-age=MaxAge, s-maxage=SMaxAge` to the http response header.
// If args is not empty, it will use the first duration as SMaxAge
func (c *Context) CacheMaxAge(age time.Duration, args ...time.Duration) {
	cache := fmt.Sprintf("public, max-age=%d", int(age.Seconds()))
	if len(args) != 0 {
		sMaxAge := args[0]
		cache += fmt.Sprintf(", s-maxage=%d", int(sMaxAge.Seconds()))
	}
	c.SetHeader(HeaderCacheControl, cache)
}

// Created sets the body to response and set the status to 201
func (c *Context) Created(body interface{}) {
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

// DisableReuse sets the context disable reuse
func (c *Context) DisableReuse() {
	atomic.StoreInt32(&c.reuseStatus, ReuseContextDisabled)
}

func (c *Context) isReuse() bool {
	return atomic.LoadInt32(&c.reuseStatus) == ReuseContextEnabled
}

// Push the target to http response
func (c *Context) Push(target string, opts *http.PushOptions) (err error) {
	if c.Response == nil {
		return ErrNilResponse
	}
	pusher, ok := c.Response.(http.Pusher)
	if !ok {
		return ErrNotSupportPush
	}
	return pusher.Push(target, opts)
}

// Elton returns the elton instance of context
func (c *Context) Elton() *Elton {
	return c.elton
}

// Pass request to another elton instance and set the context is committed
func (c *Context) Pass(another *Elton) {
	// 设置为已commit，避免当前cod继续处理
	c.Committed = true
	another.ServeHTTP(c.Response, c.Request)
}

// Pipe the reader to the response
func (c *Context) Pipe(r io.Reader) (written int64, err error) {
	c.Committed = true
	// 如果是 closer，则需要调用close函数
	closer, ok := r.(io.Closer)
	if ok {
		defer closer.Close()
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
	return c
}
