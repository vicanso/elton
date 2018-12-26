package cod

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/vicanso/errors"
)

var (
	// ErrOutOfHandlerRange out of handler range (call next over handler's size)
	ErrOutOfHandlerRange = errors.New("out of handler range")
)

type (
	// RouterInfo router's info
	RouterInfo struct {
		Method string `json:"method,omitempty"`
		Path   string `json:"path,omitempty"`
	}
	// Cod web framework instance
	Cod struct {
		Server  *http.Server
		Router  *httprouter.Router
		Routers []*RouterInfo
		// Middlewares middleware function
		Middlewares    []Handler
		errorListeners []ErrorListener
		traceListeners []TraceListener
		// ErrorHandler error handler
		ErrorHandler ErrorHandler
		// NotFoundHandler not found handler
		NotFoundHandler http.HandlerFunc
		// GenerateID generate id function
		GenerateID GenerateID
		// EnableTrace enable trace
		EnableTrace bool
		// functionInfos the function address:name map
		functionInfos map[uintptr]string
		ctxPool       sync.Pool
	}
	// TraceInfo trace's info
	TraceInfo struct {
		Name     string        `json:"name,omitempty"`
		Duration time.Duration `json:"duration,omitempty"`
	}
	// Group group router
	Group struct {
		Path        string
		HandlerList []Handler
		Cod         *Cod
	}
	// ErrorHandler error handle function
	ErrorHandler func(*Context, error)
	// GenerateID generate context id
	GenerateID func() string
	// Handler cod handle function
	Handler func(*Context) error
	// ErrorListener error listener function
	ErrorListener func(*Context, error)
	// TraceListener trace listener
	TraceListener func(*Context, []*TraceInfo)
)

// New create a cod instance
func New() *Cod {
	d := NewWithoutServer()
	s := &http.Server{
		Handler: d,
	}
	d.Server = s
	return d
}

// NewWithoutServer create a cod instance without server
func NewWithoutServer() *Cod {
	d := &Cod{
		Router:        httprouter.New(),
		Middlewares:   make([]Handler, 0),
		functionInfos: make(map[uintptr]string),
	}
	d.ctxPool.New = func() interface{} {
		return &Context{}
	}
	return d
}

// SetFunctionName set function name
func (d *Cod) SetFunctionName(fn interface{}, name string) {
	p := reflect.ValueOf(fn).Pointer()
	d.functionInfos[p] = name
}

// GetFunctionName get function name
func (d *Cod) GetFunctionName(fn interface{}) string {
	p := reflect.ValueOf(fn).Pointer()
	name := d.functionInfos[p]
	if name != "" {
		return name
	}
	return runtime.FuncForPC(p).Name()
}

// ListenAndServe listen and serve for http server
func (d *Cod) ListenAndServe(addr string) error {
	if d.Server == nil {
		panic("server is not initialized")
	}
	d.Server.Addr = addr
	return d.Server.ListenAndServe()
}

// Serve serve for http server
func (d *Cod) Serve(l net.Listener) error {
	if d.Server == nil {
		panic("server is not initialized")
	}
	return d.Server.Serve(l)
}

// Close close the http server
func (d *Cod) Close() error {
	return d.Server.Close()
}

// ServeHTTP http handler
func (d *Cod) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	fn, params, _ := d.Router.Lookup(req.Method, req.URL.Path)
	if fn != nil {
		fn(resp, req, params)
		return
	}
	// 404处理
	d.NotFound(resp, req)
}

// fillContext fill the context
func (d *Cod) fillContext(c *Context, resp http.ResponseWriter, req *http.Request) {
	c.Reset()
	c.Request = req
	c.Response = resp
	if resp != nil {
		c.Headers = resp.Header()
	}
}

// Handle add http handle function
func (d *Cod) Handle(method, path string, handlerList ...Handler) {
	for _, fn := range handlerList {
		name := d.GetFunctionName(fn)
		d.SetFunctionName(fn, name)
	}

	if d.Routers == nil {
		d.Routers = make([]*RouterInfo, 0)
	}
	d.Routers = append(d.Routers, &RouterInfo{
		Method: method,
		Path:   path,
	})
	d.Router.Handle(method, path, func(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
		c := d.ctxPool.Get().(*Context)
		d.fillContext(c, resp, req)
		c.Params = make(map[string]string)
		for _, item := range params {
			c.Params[item.Key] = item.Value
		}
		if d.GenerateID != nil {
			c.ID = d.GenerateID()
		}
		c.Route = path
		c.cod = d
		mids := d.Middlewares
		maxMid := len(mids)
		maxNext := maxMid + len(handlerList)
		index := -1
		var traceInfos []*TraceInfo
		if d.EnableTrace {
			traceInfos = make([]*TraceInfo, maxNext)
		}
		c.Next = func() error {
			index++
			var fn Handler
			// 如果调用过多的next，则会导致panic
			if index >= maxNext {
				panic(ErrOutOfHandlerRange)
			}
			// 如果已执行完公共添加的中间件，执行handler list
			if index >= maxMid {
				fn = handlerList[index-maxMid]
			} else {
				fn = mids[index]
			}
			if traceInfos == nil {
				return fn(c)
			}
			startedAt := time.Now()
			i := index
			err := fn(c)
			traceInfos[i] = &TraceInfo{
				Name:     d.GetFunctionName(fn),
				Duration: time.Since(startedAt),
			}
			return err
		}
		err := c.Next()
		if err != nil {
			d.EmitError(c, err)
			d.Error(c, err)
		} else if !c.Committed {
			resp.WriteHeader(c.StatusCode)
			_, responseErr := resp.Write(c.BodyBytes)
			if responseErr != nil {
				d.EmitError(c, responseErr)
			}
		}
		if traceInfos != nil {
			d.EmitTrace(c, traceInfos)
		}
		d.ctxPool.Put(c)
	})
}

// GET add http get method handle
func (d *Cod) GET(path string, handlerList ...Handler) {
	d.Handle(http.MethodGet, path, handlerList...)
}

// POST add http post method handle
func (d *Cod) POST(path string, handlerList ...Handler) {
	d.Handle(http.MethodPost, path, handlerList...)
}

// PUT add http put method handle
func (d *Cod) PUT(path string, handlerList ...Handler) {
	d.Handle(http.MethodPut, path, handlerList...)
}

// PATCH add http patch method handle
func (d *Cod) PATCH(path string, handlerList ...Handler) {
	d.Handle(http.MethodPatch, path, handlerList...)
}

// DELETE add http delete method handle
func (d *Cod) DELETE(path string, handlerList ...Handler) {
	d.Handle(http.MethodDelete, path, handlerList...)
}

// HEAD add http head method handle
func (d *Cod) HEAD(path string, handlerList ...Handler) {
	d.Handle(http.MethodHead, path, handlerList...)
}

// OPTIONS add http options method handle
func (d *Cod) OPTIONS(path string, handlerList ...Handler) {
	d.Handle(http.MethodOptions, path, handlerList...)
}

// TRACE add http trace method handle
func (d *Cod) TRACE(path string, handlerList ...Handler) {
	d.Handle(http.MethodTrace, path, handlerList...)
}

// ALL add http all method handle
func (d *Cod) ALL(path string, handlerList ...Handler) {
	for _, method := range methods {
		d.Handle(method, path, handlerList...)
	}
}

// Group create a http handle group
func (d *Cod) Group(path string, handlerList ...Handler) (g *Group) {
	return &Group{
		Cod:         d,
		Path:        path,
		HandlerList: handlerList,
	}
}

// Use add middleware function handle
func (d *Cod) Use(handlerList ...Handler) {
	for _, fn := range handlerList {
		name := d.GetFunctionName(fn)
		d.SetFunctionName(fn, name)
	}
	d.Middlewares = append(d.Middlewares, handlerList...)
}

// NotFound not found handle
func (d *Cod) NotFound(resp http.ResponseWriter, req *http.Request) {
	if d.NotFoundHandler != nil {
		d.NotFoundHandler(resp, req)
		return
	}
	resp.WriteHeader(http.StatusNotFound)
	resp.Write([]byte("Not found"))
}

// Error error handle
func (d *Cod) Error(c *Context, err error) {
	if d.ErrorHandler != nil {
		d.ErrorHandler(c, err)
		return
	}
	resp := c.Response
	he, ok := err.(*errors.HTTPError)
	if ok {
		resp.WriteHeader(he.StatusCode)
		resp.Write([]byte(he.Error()))
	} else {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(err.Error()))
	}
}

// EmitError emit error function
func (d *Cod) EmitError(c *Context, err error) {
	lns := d.errorListeners
	for _, ln := range lns {
		ln(c, err)
	}
}

// OnError on error function
func (d *Cod) OnError(ln ErrorListener) {
	if d.errorListeners == nil {
		d.errorListeners = make([]ErrorListener, 0)
	}
	d.errorListeners = append(d.errorListeners, ln)
}

// EmitTrace emit trace
func (d *Cod) EmitTrace(c *Context, infos []*TraceInfo) {
	lns := d.traceListeners
	for _, ln := range lns {
		ln(c, infos)
	}
}

// OnTrace on trace function
func (d *Cod) OnTrace(ln TraceListener) {
	if d.traceListeners == nil {
		d.traceListeners = make([]TraceListener, 0)
	}
	d.traceListeners = append(d.traceListeners, ln)
}

func (g *Group) merge(s2 []Handler) []Handler {
	s1 := g.HandlerList
	fns := make([]Handler, len(s1)+len(s2))
	copy(fns, s1)
	copy(fns[len(s1):], s2)
	return fns
}

// GET add group http get method handler
func (g *Group) GET(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.Cod.GET(p, fns...)
}

// POST add group http post method handler
func (g *Group) POST(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.Cod.POST(p, fns...)
}

// PUT add group http put method handler
func (g *Group) PUT(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.Cod.PUT(p, fns...)
}

// PATCH add group http patch method handler
func (g *Group) PATCH(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.Cod.PATCH(p, fns...)
}

// DELETE add group http delete method handler
func (g *Group) DELETE(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.Cod.DELETE(p, fns...)
}

// HEAD add group http head method handler
func (g *Group) HEAD(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.Cod.HEAD(p, fns...)
}

// OPTIONS add group http options method handler
func (g *Group) OPTIONS(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.Cod.OPTIONS(p, fns...)
}

// TRACE add group http trace method handler
func (g *Group) TRACE(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.Cod.TRACE(p, fns...)
}

// ALL add group http all method handler
func (g *Group) ALL(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.Cod.ALL(p, fns...)
}

// GenerateETag generate eTag
func GenerateETag(buf []byte) string {
	size := len(buf)
	if size == 0 {
		return "\"0-2jmj7l5rSw0yVb_vlWAYkK_YBwk=\""
	}
	h := sha1.New()
	h.Write(buf)
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("\"%x-%s\"", size, hash)
}
