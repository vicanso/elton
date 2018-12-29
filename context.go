package cod

import (
	"net"
	"net/http"
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
		// IgnoreNext ignore next middleware function
		IgnoreNext bool
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
	c.Headers = nil
	c.Committed = false
	c.ID = ""
	c.Route = ""
	c.Next = nil
	c.Params = nil
	c.StatusCode = 0
	c.Body = nil
	c.BodyBytes = nil
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
	if code < MinRedirectCode || code > MaxRedirectCode {
		err = ErrInvalidRedirect
		return
	}
	c.StatusCode = code
	c.SetHeader(HeaderLocation, url)
	return
}

// Set store the value in the context
func (c *Context) Set(key string, value interface{}) {
	if c.m == nil {
		c.m = make(map[string]interface{})
	}
	c.m[key] = value
}

// GetRequestHeader get from http request header
func (c *Context) GetRequestHeader(key string) string {
	return c.Request.Header.Get(key)
}

// Header get headers of http response
func (c *Context) Header() http.Header {
	return c.Headers
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

// Get get the value from context
func (c *Context) Get(key string) interface{} {
	if c.m == nil {
		return nil
	}
	return c.m[key]
}

// NoContent no content for response
func (c *Context) NoContent() {
	c.StatusCode = http.StatusNoContent
	c.Body = nil
	c.BodyBytes = nil
}

// NotModified response not modified
func (c *Context) NotModified() {
	c.StatusCode = http.StatusNotModified
	c.Body = nil
	c.BodyBytes = nil
}

// NoCache set http response no cache
func (c *Context) NoCache() {
	c.SetHeader(HeaderCacheControl, "no-cache, max-age=0")
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
