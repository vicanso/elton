package cod

import (
	"net"
	"net/http"
	"strings"
	"sync"
)

type (
	// Context cod context
	Context struct {
		Request  *http.Request
		Response http.ResponseWriter
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
		// RequestBody http request body
		RequestBody []byte
		// store for context
		m map[string]interface{}
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
	c.m = nil
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

// Redirect redirect the http request
func (c *Context) Redirect() {
	// TODO 设置重定向
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

// SetHeader set the http response header
func (c *Context) SetHeader(key, value string) {
	c.Response.Header().Set(key, value)
}

// Cookie get cookie from http request
func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

// SetCookie set the cookie for the response
func (c *Context) SetCookie(cookie *http.Cookie) {
	c.Response.Header().Add(HeaderSetCookie, cookie.String())
}

// Get get the value
func (c *Context) Get(key string) interface{} {
	if c.m == nil {
		return nil
	}
	return c.m[key]
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
