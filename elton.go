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
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vicanso/hes"
	intranetip "github.com/vicanso/intranet-ip"
)

const (
	// StatusRunning running status
	StatusRunning = iota
	// StatusClosing closing status
	StatusClosing
	// StatusClosed closed status
	StatusClosed
)

type (
	// Skipper check for skip middleware
	Skipper func(c *Context) bool
	// RouterInfo router's info
	RouterInfo struct {
		Method string `json:"method,omitempty"`
		Route  string `json:"route,omitempty"`
	}
	// Elton web framework instance
	Elton struct {
		// Server http server
		Server *http.Server
		// ErrorHandler set the function for error handler
		ErrorHandler ErrorHandler
		// NotFoundHandler set the function for not found handler
		NotFoundHandler http.HandlerFunc
		// MethodNotAllowedHandler set the function for method not allowed handler
		MethodNotAllowedHandler http.HandlerFunc
		// GenerateID generate id function, will use it to create context's id
		GenerateID GenerateID
		// EnableTrace enable trace
		EnableTrace bool
		// SignedKeys signed keys
		SignedKeys SignedKeysGenerator

		// status of elton
		status int32
		// route tree
		tree *node
		// routers all router infos
		routers []*RouterInfo
		// middlewares middleware function
		middlewares []Handler
		// preMiddlewares pre middleware function
		preMiddlewares []PreHandler
		errorListeners []ErrorListener
		traceListeners []TraceListener
		// functionInfos the function address:name map
		functionInfos map[uintptr]string
		ctxPool       sync.Pool
	}

	// Router router
	Router struct {
		Method     string    `json:"method,omitempty"`
		Path       string    `json:"path,omitempty"`
		HandleList []Handler `json:"-"`
	}
	// Group group router
	Group struct {
		Path        string
		HandlerList []Handler
		routers     []*Router
	}
	// ErrorHandler error handle function
	ErrorHandler func(*Context, error)
	// GenerateID generate context id
	GenerateID func() string
	// Handler elton handle function
	Handler func(*Context) error
	// ErrorListener error listener function
	ErrorListener func(*Context, error)
	// TraceListener trace listener
	TraceListener func(*Context, TraceInfos)
	// PreHandler pre handler
	PreHandler func(*http.Request)
)

var _ http.Handler = (*Elton)(nil)

// DefaultSkipper default skipper function
func DefaultSkipper(c *Context) bool {
	return c.Committed
}

// New returns a new elton instance
func New() *Elton {
	e := NewWithoutServer()
	s := &http.Server{
		Handler: e,
	}
	e.Server = s
	return e
}

// NewWithoutServer returns a new elton instance without http server
func NewWithoutServer() *Elton {
	e := &Elton{
		tree:          new(node),
		functionInfos: make(map[uintptr]string),
	}
	e.ctxPool.New = func() interface{} {
		return &Context{
			elton:  e,
			Params: new(RouteParams),
		}
	}
	return e
}

// NewGroup returns a new  router group
func NewGroup(path string, handlerList ...Handler) *Group {
	return &Group{
		Path:        path,
		HandlerList: handlerList,
	}
}

// IsIntranet judgets whether the ip is intranet
func IsIntranet(ip string) bool {
	return intranetip.Is(net.ParseIP(ip))
}

// SetFunctionName sets the name of handler function,
// it will use to http timing
func (e *Elton) SetFunctionName(fn interface{}, name string) {
	p := reflect.ValueOf(fn).Pointer()
	e.functionInfos[p] = name
}

// GetFunctionName return the name of handler function
func (e *Elton) GetFunctionName(fn interface{}) string {
	p := reflect.ValueOf(fn).Pointer()
	name := e.functionInfos[p]
	if name != "" {
		return name
	}
	return runtime.FuncForPC(p).Name()
}

// ListenAndServe listens the addr and serve http,
// it will throw panic if the server of elton is nil.
func (e *Elton) ListenAndServe(addr string) error {
	if e.Server == nil {
		panic(errors.New("server is not initialized"))
	}
	e.Server.Addr = addr
	return e.Server.ListenAndServe()
}

// ListenAndServeTLS listens the addr and server https,
// it will throw panic if the server of elton is nil.
func (e *Elton) ListenAndServeTLS(addr, certFile, keyFile string) error {
	if e.Server == nil {
		panic(errors.New("server is not initialized"))
	}
	e.Server.Addr = addr
	return e.Server.ListenAndServeTLS(certFile, keyFile)
}

// Serve serves http server,
// it will throw panic if the server of elton is nil.
func (e *Elton) Serve(l net.Listener) error {
	if e.Server == nil {
		panic(errors.New("server is not initialized"))
	}
	return e.Server.Serve(l)
}

// Close closes the http server
func (e *Elton) Close() error {
	return e.Server.Close()
}

// Shutdown shotdowns the http server
func (e *Elton) Shutdown() error {
	return e.Server.Shutdown(context.Background())
}

// GracefulClose closes the http server graceful.
// It sets the status to be closing and delay to close.
func (e *Elton) GracefulClose(delay time.Duration) error {
	atomic.StoreInt32(&e.status, StatusClosing)
	time.Sleep(delay)
	atomic.StoreInt32(&e.status, StatusClosed)
	return e.Shutdown()
}

// GetStatus returns status of elton
func (e *Elton) GetStatus() int32 {
	return atomic.LoadInt32(&e.status)
}

// Closing judge the status whether is closing
func (e *Elton) Closing() bool {
	return e.GetStatus() == StatusClosing
}

// Running judge the status whether is running
func (e *Elton) Running() bool {
	return e.GetStatus() == StatusRunning
}

// ServeHTTP http handler
func (e *Elton) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	status := e.GetStatus()
	// 非运行中的状态
	if status != StatusRunning {
		resp.WriteHeader(http.StatusServiceUnavailable)
		_, err := resp.Write([]byte(fmt.Sprintf("service is not available, status is %d", status)))
		if err != nil {
			e.emitError(resp, req, err)
		}
		return
	}
	for _, preHandler := range e.preMiddlewares {
		preHandler(req)
	}

	c := e.ctxPool.Get().(*Context)
	c.Reset()
	methodType := methodTypeMap[req.Method]
	rn := e.tree.findRoute(methodType, req.URL.Path, c.Params)
	if rn == nil {
		if c.Params.methodNotAllowed {
			e.methodNotAllowed(resp, req)
		} else {
			// 404处理
			e.notFound(resp, req)
		}
		// not found 与method not allowed所有context都可复用
		e.ctxPool.Put(c)
		return
	}
	c.Request = req
	c.Response = resp

	if e.GenerateID != nil {
		c.ID = e.GenerateID()
	}

	rn.endpoints[methodType].handler(c)
	if c.isReuse() {
		e.ctxPool.Put(c)
	}
}

// GetRouters returns routers of elton
func (e *Elton) GetRouters() []RouterInfo {
	routers := make([]RouterInfo, len(e.routers))
	for index, r := range e.routers {
		routers[index] = *r
	}
	return routers
}

// Handle adds http handle function
func (e *Elton) Handle(method, path string, handlerList ...Handler) *Elton {
	for _, fn := range handlerList {
		name := e.GetFunctionName(fn)
		e.SetFunctionName(fn, name)
	}

	if e.routers == nil {
		e.routers = make([]*RouterInfo, 0)
	}
	e.routers = append(e.routers, &RouterInfo{
		Method: method,
		Route:  path,
	})
	e.tree.InsertRoute(methodTypeMap[method], path, func(c *Context) {
		c.Route = path
		mids := e.middlewares
		maxMid := len(mids)
		maxNext := maxMid + len(handlerList)
		index := -1
		var trace *Trace
		if e.EnableTrace {
			trace = NewTrace()
			c.WithContext(context.WithValue(c.Context(), ContextTraceKey, trace))
		}
		c.Next = func() error {
			// 如果已设置响应数据，则不再执行后续的中间件
			if c.Committed {
				return nil
			}
			index++
			var fn Handler
			// 如果调用过多的next，则直接返回
			if index >= maxNext {
				return nil
			}

			// 如果已执行完公共添加的中间件，执行handler list
			if index >= maxMid {
				fn = handlerList[index-maxMid]
			} else {
				fn = mids[index]
			}
			if trace == nil {
				return fn(c)
			}
			fnName := e.GetFunctionName(fn)
			// 如果函数名字为 - ，则跳过
			if fnName == "-" {
				return fn(c)
			}
			startedAt := time.Now()

			traceInfo := &TraceInfo{
				Name:       fnName,
				Middleware: true,
			}
			// 先添加至slice中，保证顺序
			trace.Add(traceInfo)
			err := fn(c)
			// 完成后计算时长（前面的中间件包括后面中间件的处理时长）
			traceInfo.Duration = time.Since(startedAt)
			return err
		}
		err := c.Next()
		if trace != nil {
			trace.Calculate()
			e.EmitTrace(c, trace.Infos)
		}
		if err != nil {
			e.EmitError(c, err)
		}
		// 如果已commit 表示返回数据已设置，无需处理
		if c.Committed {
			return
		}
		c.Committed = true
		// 如果出错则触发出错处理，返回
		if err != nil {
			e.error(c, err)
			return
		}
		// 需要在设置status code之前设置响应长度
		if c.BodyBuffer != nil {
			c.SetHeader(HeaderContentLength, strconv.Itoa(c.BodyBuffer.Len()))
		}
		if c.StatusCode != 0 {
			c.Response.WriteHeader(c.StatusCode)
		}
		if c.BodyBuffer != nil {
			_, responseErr := c.Response.Write(c.BodyBuffer.Bytes())
			if responseErr != nil {
				e.EmitError(c, responseErr)
			}
		} else if c.IsReaderBody() {
			r, _ := c.Body.(io.Reader)
			_, pipeErr := c.Pipe(r)
			if pipeErr != nil {
				e.EmitError(c, pipeErr)
			}
		}
	})
	return e
}

// GET adds http get method handle
func (e *Elton) GET(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodGet, path, handlerList...)
}

// POST adds http post method handle
func (e *Elton) POST(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodPost, path, handlerList...)
}

// PUT adds http put method handle
func (e *Elton) PUT(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodPut, path, handlerList...)
}

// PATCH adds http patch method handle
func (e *Elton) PATCH(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodPatch, path, handlerList...)
}

// DELETE adds http delete method handle
func (e *Elton) DELETE(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodDelete, path, handlerList...)
}

// HEAD adds http head method handle
func (e *Elton) HEAD(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodHead, path, handlerList...)
}

// OPTIONS adds http options method handle
func (e *Elton) OPTIONS(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodOptions, path, handlerList...)
}

// TRACE adds http trace method handle
func (e *Elton) TRACE(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodTrace, path, handlerList...)
}

// ALL adds http all method handle
func (e *Elton) ALL(path string, handlerList ...Handler) *Elton {
	for _, method := range methods {
		e.Handle(method, path, handlerList...)
	}
	return e
}

// Use adds middleware handler function to elton's middleware list
func (e *Elton) Use(handlerList ...Handler) *Elton {
	if e.middlewares == nil {
		e.middlewares = make([]Handler, 0)
	}
	for _, fn := range handlerList {
		name := e.GetFunctionName(fn)
		e.SetFunctionName(fn, name)
	}
	e.middlewares = append(e.middlewares, handlerList...)
	return e
}

// UseWithName adds middleware and set handler function's name
func (e *Elton) UseWithName(handler Handler, name string) *Elton {
	e.SetFunctionName(handler, name)
	return e.Use(handler)
}

// Pre adds pre middleware function handler to elton's pre middleware list
func (e *Elton) Pre(handlerList ...PreHandler) *Elton {
	if e.preMiddlewares == nil {
		e.preMiddlewares = make([]PreHandler, 0)
	}
	e.preMiddlewares = append(e.preMiddlewares, handlerList...)
	return e
}

// notFound not found handle
func (e *Elton) notFound(resp http.ResponseWriter, req *http.Request) *Elton {
	if e.NotFoundHandler != nil {
		e.NotFoundHandler(resp, req)
		return e
	}
	resp.WriteHeader(http.StatusNotFound)
	_, err := resp.Write([]byte("Not Found"))
	if err != nil {
		e.emitError(resp, req, err)
	}
	return e
}

// methodNotAllowed method not allowed handle
func (e *Elton) methodNotAllowed(resp http.ResponseWriter, req *http.Request) *Elton {
	if e.MethodNotAllowedHandler != nil {
		e.MethodNotAllowedHandler(resp, req)
		return e
	}
	resp.WriteHeader(http.StatusMethodNotAllowed)
	_, err := resp.Write([]byte("Method Not Allowed"))
	if err != nil {
		e.emitError(resp, req, err)
	}
	return e
}

// error error handle
func (e *Elton) error(c *Context, err error) *Elton {
	// 出错时清除部分响应头
	for _, key := range []string{
		HeaderETag,
		HeaderLastModified,
		HeaderContentEncoding,
		HeaderContentLength,
	} {
		c.SetHeader(key, "")
	}
	if e.ErrorHandler != nil {
		e.ErrorHandler(c, err)
		return e
	}

	resp := c.Response
	he, ok := err.(*hes.Error)
	status := http.StatusInternalServerError
	message := err.Error()
	if ok {
		status = he.StatusCode
		message = he.Error()
	}
	resp.WriteHeader(status)
	_, err = resp.Write([]byte(message))
	if err != nil {
		e.EmitError(c, err)
	}
	return e
}

// EmitError emits an error event, it will call the listen functions of error event
func (e *Elton) EmitError(c *Context, err error) *Elton {
	lns := e.errorListeners
	for _, ln := range lns {
		ln(c, err)
	}
	return e
}

func (e *Elton) emitError(resp http.ResponseWriter, req *http.Request, err error) {
	e.EmitError(&Context{
		Request:  req,
		Response: resp,
	}, err)
}

// OnError adds listen to error event
func (e *Elton) OnError(ln ErrorListener) *Elton {
	if e.errorListeners == nil {
		e.errorListeners = make([]ErrorListener, 0)
	}
	e.errorListeners = append(e.errorListeners, ln)
	return e
}

// EmitTrace emits a trace event, it will call the listen functions of trace event
func (e *Elton) EmitTrace(c *Context, infos TraceInfos) *Elton {
	lns := e.traceListeners
	for _, ln := range lns {
		ln(c, infos)
	}
	return e
}

// OnTrace adds listen to trace event
func (e *Elton) OnTrace(ln TraceListener) *Elton {
	if e.traceListeners == nil {
		e.traceListeners = make([]TraceListener, 0)
	}
	e.traceListeners = append(e.traceListeners, ln)
	return e
}

// AddGroup adds the group to elton
func (e *Elton) AddGroup(groups ...*Group) *Elton {
	for _, g := range groups {
		for _, r := range g.routers {
			e.Handle(r.Method, r.Path, r.HandleList...)
		}
	}
	return e
}

func (g *Group) merge(s2 []Handler) []Handler {
	s1 := g.HandlerList
	fns := make([]Handler, len(s1)+len(s2))
	copy(fns, s1)
	copy(fns[len(s1):], s2)
	return fns
}

func (g *Group) add(method, path string, handlerList ...Handler) {
	if g.routers == nil {
		g.routers = make([]*Router, 0, 5)
	}
	g.routers = append(g.routers, &Router{
		Method:     method,
		Path:       path,
		HandleList: handlerList,
	})
}

// GET adds http get method handler to group
func (g *Group) GET(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodGet, p, fns...)
}

// POST adds http post method handler to group
func (g *Group) POST(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodPost, p, fns...)
}

// PUT adds http put method handler to group
func (g *Group) PUT(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodPut, p, fns...)
}

// PATCH adds http patch method handler to group
func (g *Group) PATCH(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodPatch, p, fns...)
}

// DELETE adds http delete method handler to group
func (g *Group) DELETE(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodDelete, p, fns...)
}

// HEAD adds http head method handler to group
func (g *Group) HEAD(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodHead, p, fns...)
}

// OPTIONS adds http options method handler to group
func (g *Group) OPTIONS(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodOptions, p, fns...)
}

// TRACE adds http trace method handler to group
func (g *Group) TRACE(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodTrace, p, fns...)
}

// ALL adds http all methods handler to group
func (g *Group) ALL(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	for _, method := range methods {
		g.add(method, p, fns...)
	}
}

// Compose composes handler list as a handler
func Compose(handlerList ...Handler) Handler {
	max := len(handlerList)
	if max == 0 {
		panic(errors.New("handler function is required"))
	}
	return func(c *Context) (err error) {
		// 保存原有的next函数
		originalNext := c.Next
		index := -1
		// 新创建一个next的调用链
		c.Next = func() error {
			index++
			// 如果已执行成所有的next，则转回原有的调用链
			if index >= max {
				c.Next = originalNext
				return c.Next()
			}
			return handlerList[index](c)
		}
		return c.Next()
	}
}

// copy from io.ReadAll
// ReadAll reads from r until an error or EOF and returns the data it read.
// A successful call returns err == nil, not err == EOF. Because ReadAll is
// defined to read from src until EOF, it does not treat an EOF from Read
// as an error to be reported.
func ReadAllInitCap(r io.Reader, initCap int) ([]byte, error) {
	if initCap <= 0 {
		initCap = 512
	}
	b := make([]byte, 0, initCap)
	for {
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
	}
}
