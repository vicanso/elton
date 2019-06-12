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
	"bytes"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/vicanso/keygrip"
)

type (
	// Context cod context
	Context struct {
		Request  *http.Request
		Response http.ResponseWriter
		// Headers http response's header
		Headers http.Header
		// Committed commit the data to response
		Committed bool
		// ID context id
		ID string
		// Route route path
		Route string
		// Next next function
		Next func() error
		// Params uri params
		Params map[string]string
		// RawParams http router params
		RawParams httprouter.Params
		// StatusCode http response's status code
		StatusCode int
		// Body http response's body
		Body interface{}
		// BodyBuffer http response's body buffer
		BodyBuffer *bytes.Buffer
		// RequestBody http request body
		RequestBody []byte
		// store for context
		m map[interface{}]interface{}
		// realIP the real ip
		realIP string
		// clientIP the clint ip
		clientIP string
		// cod instance
		cod *Cod
		// reuseDisabled reuse disabled
		reuseDisabled bool
	}
)

const (
	sig = ".sig"
)

// Reset reset context
func (c *Context) Reset() {
	c.Request = nil
	c.Response = nil
	c.Headers = nil
	c.Committed = false
	c.ID = ""
	c.Route = ""
	c.Next = nil
	c.Params = nil
	c.RawParams = nil
	c.StatusCode = 0
	c.Body = nil
	c.BodyBuffer = nil
	c.RequestBody = nil
	c.m = nil
	c.realIP = ""
	c.clientIP = ""
	c.cod = nil
	c.reuseDisabled = false
}

// RemoteAddr get remote address
func (c *Context) RemoteAddr() string {
	remoteAddr, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	return remoteAddr
}

// RealIP get the real ip
func (c *Context) RealIP() string {
	if c.realIP != "" {
		return c.realIP
	}
	ip := c.GetRequestHeader(HeaderXForwardedFor)
	if ip != "" {
		c.realIP = strings.TrimSpace(strings.Split(ip, ",")[0])
		return c.realIP
	}
	c.realIP = c.GetRequestHeader(HeaderXRealIP)
	if c.realIP != "" {
		return c.realIP
	}
	c.realIP = c.RemoteAddr()
	return c.realIP
}

// ClientIP get the client ip
// get the first public ip from x-forwarded-for --> x-real-ip
// if not found, then get remote addr
func (c *Context) ClientIP() string {
	if c.clientIP != "" {
		return c.clientIP
	}
	ip := c.GetRequestHeader(HeaderXForwardedFor)
	if ip != "" {
		for _, value := range strings.Split(ip, ",") {
			v := strings.TrimSpace(value)
			if !IsPrivateIP(net.ParseIP(v)) {
				c.clientIP = v
				return c.clientIP
			}
		}
	}
	ip = c.GetRequestHeader(HeaderXRealIP)
	if ip != "" {
		if !IsPrivateIP(net.ParseIP(ip)) {
			c.clientIP = ip
			return c.clientIP
		}
	}
	// 如果都不符合，只能直接取real ip
	c.clientIP = c.RemoteAddr()
	return c.clientIP
}

// Param get the param value
func (c *Context) Param(name string) string {
	if c.Params == nil {
		return ""
	}
	return c.Params[name]
}

// QueryParam get the query value
func (c *Context) QueryParam(name string) string {
	values := c.Request.URL.Query()[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// Query get the query map.
// It will return map[string]string, not the same as url.Values
func (c *Context) Query() map[string]string {
	values := c.Request.URL.Query()
	if len(values) == 0 {
		return nil
	}
	m := make(map[string]string)
	for key, values := range values {
		m[key] = values[0]
	}
	return m
}

// Redirect redirect the http request
func (c *Context) Redirect(code int, url string) (err error) {
	if code < MinRedirectCode || code > MaxRedirectCode {
		err = ErrInvalidRedirect
		return
	}
	c.StatusCode = code
	c.SetHeader(HeaderLocation, url)
	return
}

// Set store the value in the context
func (c *Context) Set(key, value interface{}) {
	if c.m == nil {
		c.m = make(map[interface{}]interface{}, 5)
	}
	c.m[key] = value
}

// Get get the value from context
func (c *Context) Get(key interface{}) interface{} {
	if c.m == nil {
		return nil
	}
	return c.m[key]
}

// GetRequestHeader get from http request header
func (c *Context) GetRequestHeader(key string) string {
	return c.Request.Header.Get(key)
}

// SetRequestHeader set http request header
func (c *Context) SetRequestHeader(key, value string) {
	h := c.Request.Header
	if value == "" {
		h.Del(key)
		return
	}
	h.Set(key, value)
}

// AddRequestHeader add http request header
func (c *Context) AddRequestHeader(key, value string) {
	c.Request.Header.Add(key, value)
}

// Header get headers of http response
func (c *Context) Header() http.Header {
	return c.Headers
}

// WriteHeader set the http status code
func (c *Context) WriteHeader(statusCode int) {
	c.StatusCode = statusCode
}

// Write write the response body
func (c *Context) Write(buf []byte) (int, error) {
	if c.BodyBuffer == nil {
		c.BodyBuffer = new(bytes.Buffer)
	}
	return c.BodyBuffer.Write(buf)
}

// GetHeader get header from http response
func (c *Context) GetHeader(key string) string {
	return c.Headers.Get(key)
}

// SetHeader set header to the http response
func (c *Context) SetHeader(key, value string) {
	if value == "" {
		c.Headers.Del(key)
		return
	}
	c.Headers.Set(key, value)
}

// AddHeader add header to the http response
func (c *Context) AddHeader(key, value string) {
	c.Headers.Add(key, value)
}

// ResetHeader reset response header
func (c *Context) ResetHeader() {
	h := c.Header()
	for k := range h {
		h.Del(k)
	}
}

// Cookie get cookie from http request
func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

// AddCookie add the cookie to the response
func (c *Context) AddCookie(cookie *http.Cookie) error {
	c.AddHeader(HeaderSetCookie, cookie.String())
	return nil
}

func (c *Context) getKeys() []string {
	d := c.Cod(nil)
	if d == nil || d.SignedKeys == nil {
		return nil
	}
	return d.SignedKeys.GetKeys()
}

// SignedCookie get signed cookie from http request
func (c *Context) SignedCookie(name string) (cookie *http.Cookie, err error) {
	cookie, err = c.Cookie(name)
	if err != nil {
		return
	}
	keys := c.getKeys()
	// 如果没有配置keys，则认为cookie符合
	if len(keys) == 0 {
		return
	}

	sc, err := c.Cookie(name + sig)
	if err != nil {
		cookie = nil
		return
	}
	kg := keygrip.New(keys)
	// 如果校验不符合，则与查找不到cookie 一样
	if !kg.Verify([]byte(cookie.Value), []byte(sc.Value)) {
		cookie = nil
		err = http.ErrNoCookie
	}
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

// AddSignedCookie add the signed cookie to the response
func (c *Context) AddSignedCookie(cookie *http.Cookie) (err error) {
	err = c.AddCookie(cookie)
	if err != nil {
		return
	}
	sc := cloneCookie(cookie)
	sc.Name = sc.Name + sig
	keys := c.getKeys()
	if len(keys) == 0 {
		return
	}
	kg := keygrip.New(keys)
	sc.Value = string(kg.Sign([]byte(sc.Value)))
	err = c.AddCookie(sc)
	return
}

// NoContent no content for response
func (c *Context) NoContent() {
	c.StatusCode = http.StatusNoContent
	c.Body = nil
	c.BodyBuffer = nil
	c.SetHeader(HeaderContentType, "")
	c.SetHeader(HeaderContentLength, "")
	c.SetHeader(HeaderTransferEncoding, "")
}

// NotModified response not modified
func (c *Context) NotModified() {
	c.StatusCode = http.StatusNotModified
	c.SetHeader(HeaderContentEncoding, "")
	c.SetHeader(HeaderContentType, "")
	c.Body = nil
	c.BodyBuffer = nil
}

// NoCache set http response no cache
func (c *Context) NoCache() {
	c.SetHeader(HeaderCacheControl, "no-cache")
}

// NoStore set http response no store
func (c *Context) NoStore() {
	c.SetHeader(HeaderCacheControl, "no-store")
}

// CacheMaxAge set http response to cache for max age
func (c *Context) CacheMaxAge(age string) {
	d, _ := time.ParseDuration(age)
	cache := "public, max-age=" + strconv.Itoa(int(d.Seconds()))
	c.SetHeader(HeaderCacheControl, cache)
}

// CacheSMaxAge set http response to cache for s-max age
func (c *Context) CacheSMaxAge(age, sMaxAge string) {
	d, _ := time.ParseDuration(age)
	d1, _ := time.ParseDuration(sMaxAge)
	cache := fmt.Sprintf("public, max-age=%d, s-maxage=%d", int(d.Seconds()), int(d1.Seconds()))
	c.SetHeader(HeaderCacheControl, cache)
}

// Created created for response
func (c *Context) Created(body interface{}) {
	c.StatusCode = http.StatusCreated
	c.Body = body
}

// SetContentTypeByExt set content type by file extname
func (c *Context) SetContentTypeByExt(file string) {
	ext := filepath.Ext(file)
	contentType := mime.TypeByExtension(ext)
	if contentType != "" {
		c.SetHeader(HeaderContentType, contentType)
	}
}

// DisableReuse set the context disable reuse
func (c *Context) DisableReuse() {
	c.reuseDisabled = true
}

// Push http server push
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

// Cod get cod instance
func (c *Context) Cod(d *Cod) *Cod {
	if d != nil {
		c.cod = d
	}
	return c.cod
}

// Pass pass requst to another cod
func (c *Context) Pass(another *Cod) {
	// 设置为已commit，避免当前cod继续处理
	c.Committed = true
	another.ServeHTTP(c.Response, c.Request)
}

// Pipe pie to the response
func (c *Context) Pipe(r io.Reader) (written int64, err error) {
	c.Committed = true
	// 如果是 closer，则需要调用close函数
	closer, ok := r.(io.Closer)
	if ok {
		defer closer.Close()
	}
	return io.Copy(c.Response, r)
}

// IsReaderBody check body is reader
func (c *Context) IsReaderBody() bool {
	if c.Body == nil {
		return false
	}
	_, ok := c.Body.(io.Reader)
	return ok
}

// NewContext new a context
func NewContext(resp http.ResponseWriter, req *http.Request) *Context {
	c := &Context{}
	c.Request = req
	c.Response = resp
	if resp != nil {
		c.Headers = resp.Header()
	}
	return c
}
