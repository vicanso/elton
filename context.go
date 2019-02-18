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
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
		// cod instance
		cod *Cod
	}
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
	c.StatusCode = 0
	c.Body = nil
	c.BodyBuffer = nil
	c.RequestBody = nil
	c.m = nil
	c.realIP = ""
	c.cod = nil
}

// RealIP get the real ip
func (c *Context) RealIP() string {
	if c.realIP != "" {
		return c.realIP
	}
	h := c.Request.Header
	ip := h.Get(HeaderXForwardedFor)
	if ip != "" {
		c.realIP = strings.TrimSpace(strings.Split(ip, ",")[0])
		return c.realIP
	}
	c.realIP = h.Get(HeaderXRealIp)
	if c.realIP != "" {
		return c.realIP
	}
	c.realIP, _, _ = net.SplitHostPort(c.Request.RemoteAddr)
	return c.realIP
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

// Cookie get cookie from http request
func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

// SetCookie set the cookie for the response
func (c *Context) SetCookie(cookie *http.Cookie) error {
	c.AddHeader(HeaderSetCookie, cookie.String())
	return nil
}

// NoContent no content for response
func (c *Context) NoContent() {
	c.StatusCode = http.StatusNoContent
	c.Body = nil
	c.BodyBuffer = nil
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

// Push http server push
func (c *Context) Push(target string, opts *http.PushOptions) (err error) {
	if c.Response == nil {
		return ErrNillResponse
	}
	pusher, ok := c.Response.(http.Pusher)
	if !ok {
		return nil
	}
	return pusher.Push(target, opts)
}

// Cod get cod instance
func (c *Context) Cod() *Cod {
	return c.cod
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
