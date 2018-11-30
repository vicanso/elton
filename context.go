package cod

import (
	"net/http"
	"sync"
)

type (
	// Context cod context
	Context struct {
		Request  *http.Request
		Response http.ResponseWriter
		Route    string
		Next     func() error
		Params   map[string]string
	}
)

// Reset reset context
func (c *Context) Reset() {
	c.Request = nil
	c.Response = nil
	c.Route = ""
	c.Next = nil
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
