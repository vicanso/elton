package cod

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type (
	// Context cod context
	Context struct {
		Request  *http.Request
		Response http.ResponseWriter
		// ID request id
		ID string
		// Route route path
		Route string
		// Next next function
		Next func() error
		// Params uri params
		Params map[string]string
		// Status http response's status
		Status int
		// Body http response's body
		Body interface{}
		// BodyBytes http response's body byte
		BodyBytes []byte
		// RequestBody http request body
		RequestBody []byte
		// store for context
		m map[string]interface{}
		// cod instance
		cod *Cod
	}
)

// Reset reset context
func (c *Context) Reset() {
	c.Request = nil
	c.Response = nil
	c.Route = ""
	c.Next = nil
	c.Params = nil
	c.RequestBody = nil
	c.Status = 0
	c.BodyBytes = nil
	c.m = nil
	c.ID = ""
}

// RealIP get the real ip
func (c *Context) RealIP() string {
	h := c.Request.Header
	ip := h.Get(HeaderXForwardedFor)
	if ip != "" {
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	ip = h.Get(HeaderXRealIp)
	if ip != "" {
		return ip
	}
	ip, _, _ = net.SplitHostPort(c.Request.RemoteAddr)
	return ip
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
	m := make(map[string]string)
	for key, values := range values {
		m[key] = values[0]
	}
	return m
}

// Redirect redirect the http request
func (c *Context) Redirect(code int, url string) (err error) {
	// TODO 设置重定向
	if code < MinRedirectCode || code > MaxRedirectCode {
		err = ErrInvalidRedirect
		return
	}
	c.Status = code
	c.SetHeader(HeaderLocation, url)
	return
}

// Set set the value
func (c *Context) Set(key string, value interface{}) {
	if c.m == nil {
		c.m = make(map[string]interface{})
	}
	c.m[key] = value
}

// Header get from http request header
func (c *Context) Header(key string) string {
	return c.Request.Header.Get(key)
}

// SetHeader set header to the http response
func (c *Context) SetHeader(key, value string) {
	// TODO 是否需要创建新的header来临时保存相关header
	c.Response.Header().Set(key, value)
}

// AddHeader add header to the http response
func (c *Context) AddHeader(key, value string) {
	c.Response.Header().Add(key, value)
}

// Cookie get cookie from http request
func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

// SetCookie set the cookie for the response
func (c *Context) SetCookie(cookie *http.Cookie) {
	c.AddHeader(HeaderSetCookie, cookie.String())
}

// Get get the value
func (c *Context) Get(key string) interface{} {
	if c.m == nil {
		return nil
	}
	return c.m[key]
}

// NoContent no content for response
func (c *Context) NoContent() {
	c.Status = http.StatusNoContent
	c.Body = nil
}

// NoCache set http no cache
func (c *Context) NoCache() {
	c.SetHeader(HeaderCacheControl, "no-cache, max-age=0")
}

// NoStore set http no store
func (c *Context) NoStore() {
	c.SetHeader(HeaderCacheControl, "no-store")
}

// CacheMaxAge set http cache for max age
func (c *Context) CacheMaxAge(age string) {
	d, _ := time.ParseDuration(age)
	cache := "public, max-age=" + strconv.Itoa(int(d.Seconds()))
	c.SetHeader(HeaderCacheControl, cache)
}

// Created created for response
func (c *Context) Created(body interface{}) {
	c.Status = http.StatusCreated
	c.Body = body
}

// Cod get cod instance
func (c *Context) Cod() *Cod {
	return c.cod
}

var contextPool = sync.Pool{
	New: func() interface{} {
		return &Context{}
	},
}

// NewContext new a context
func NewContext(resp http.ResponseWriter, req *http.Request) *Context {
	c := contextPool.Get().(*Context)
	c.Reset()
	c.Request = req
	c.Response = resp
	return c
}

// ReleaseContext release context
func ReleaseContext(c *Context) {
	contextPool.Put(c)
}
