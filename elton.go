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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vicanso/hes"
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
		Path   string `json:"path,omitempty"`
	}
	// Elton web framework instance
	Elton struct {
		// status of elton
		status int32
		tree   *node
		// Server http server
		Server *http.Server
		// Routers all router infos
		Routers []*RouterInfo
		// Middlewares middleware function
		Middlewares []Handler
		// PreMiddlewares pre middleware function
		PreMiddlewares []PreHandler
		errorListeners []ErrorListener
		traceListeners []TraceListener
		// ErrorHandler set the function for error handler
		ErrorHandler ErrorHandler
		// NotFoundHandler set the function for not found handler
		NotFoundHandler http.HandlerFunc
		// MethodNotAllowedHandler set the function for method not allowed handler
		MethodNotAllowedHandler http.HandlerFunc
		// GenerateID generate id function, will use it for create id for context
		GenerateID GenerateID
		// EnableTrace enable trace
		EnableTrace bool
		// SignedKeys signed keys
		SignedKeys SignedKeysGenerator
		// functionInfos the function address:name map
		functionInfos map[uintptr]string
		ctxPool       sync.Pool
	}
	// TraceInfo trace's info
	TraceInfo struct {
		Name     string        `json:"name,omitempty"`
		Duration time.Duration `json:"duration,omitempty"`
	}
	// TraceInfos trace infos
	TraceInfos []*TraceInfo
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

// DefaultSkipper default skipper function(not skip)
func DefaultSkipper(c *Context) bool {
	return c.Committed
}

// New create an elton instance
func New() *Elton {
	e := NewWithoutServer()
	s := &http.Server{
		Handler: e,
	}
	e.Server = s
	return e
}

// NewWithoutServer create an elton instance without http server
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

// NewGroup new group
func NewGroup(path string, handlerList ...Handler) *Group {
	return &Group{
		Path:        path,
		HandlerList: handlerList,
	}
}

// SetFunctionName set function name
func (e *Elton) SetFunctionName(fn interface{}, name string) {
	p := reflect.ValueOf(fn).Pointer()
	e.functionInfos[p] = name
}

// GetFunctionName get function name
func (e *Elton) GetFunctionName(fn interface{}) string {
	p := reflect.ValueOf(fn).Pointer()
	name := e.functionInfos[p]
	if name != "" {
		return name
	}
	return runtime.FuncForPC(p).Name()
}

// ListenAndServe listen and serve for http server
func (e *Elton) ListenAndServe(addr string) error {
	if e.Server == nil {
		panic(errors.New("server is not initialized"))
	}
	e.Server.Addr = addr
	return e.Server.ListenAndServe()
}

// ListenAndServeTLS listend and serve for https server
func (e *Elton) ListenAndServeTLS(addr, certFile, keyFile string) error {
	if e.Server == nil {
		panic(errors.New("server is not initialized"))
	}
	e.Server.Addr = addr
	return e.Server.ListenAndServeTLS(certFile, keyFile)
}

// Serve serve for http server
func (e *Elton) Serve(l net.Listener) error {
	if e.Server == nil {
		panic(errors.New("server is not initialized"))
	}
	return e.Server.Serve(l)
}

// Close close the http server
func (e *Elton) Close() error {
	return e.Server.Close()
}

// GracefulClose graceful close the http server.
// It sets the status to be closing and delay to close.
func (e *Elton) GracefulClose(delay time.Duration) error {
	atomic.StoreInt32(&e.status, StatusClosing)
	time.Sleep(delay)
	atomic.StoreInt32(&e.status, StatusClosed)
	return e.Close()
}

// GetStatus get status of elton
func (e *Elton) GetStatus() int32 {
	return atomic.LoadInt32(&e.status)
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
	for _, preHandler := range e.PreMiddlewares {
		preHandler(req)
	}

	c := e.ctxPool.Get().(*Context)
	c.Reset()
	method := methodMap[req.Method]
	rn := e.tree.findRoute(method, req.URL.Path, c.Params)
	if rn == nil {
		if c.Params.methodNotAllowed {
			e.MethodNotAllowed(resp, req)
		} else {
			// 404处理
			e.NotFound(resp, req)
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

	rn.endpoints[method].handler(c)
	if c.isReuse() {
		e.ctxPool.Put(c)
	}
}

// Handle add http handle function
func (e *Elton) Handle(method, path string, handlerList ...Handler) *Elton {
	for _, fn := range handlerList {
		name := e.GetFunctionName(fn)
		e.SetFunctionName(fn, name)
	}

	if e.Routers == nil {
		e.Routers = make([]*RouterInfo, 0)
	}
	e.Routers = append(e.Routers, &RouterInfo{
		Method: method,
		Path:   path,
	})
	e.tree.InsertRoute(methodMap[method], path, func(c *Context) {
		c.Route = path
		mids := e.Middlewares
		maxMid := len(mids)
		maxNext := maxMid + len(handlerList)
		index := -1
		var traceInfos TraceInfos
		if e.EnableTrace {
			// TODO 复用tracInfos
			traceInfos = make(TraceInfos, 0, maxNext)
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
			if traceInfos == nil {
				return fn(c)
			}
			fnName := e.GetFunctionName(fn)
			// 如果函数名字为 - ，则跳过
			if fnName == "-" {
				return fn(c)
			}
			startedAt := time.Now()

			traceInfo := &TraceInfo{
				Name: fnName,
			}
			// 先添加至slice中，保证顺序
			traceInfos = append(traceInfos, traceInfo)
			err := fn(c)
			// 完成后计算时长（前面的中间件包括后面中间件的处理时长）
			traceInfo.Duration = time.Since(startedAt)
			return err
		}
		err := c.Next()
		if traceInfos != nil {
			max := len(traceInfos)
			for i, traceInfo := range traceInfos {
				if i < max-1 {
					// 计算真实耗时（不包括后面中间件处理时长）
					traceInfo.Duration -= traceInfos[i+1].Duration
				}
			}
			e.EmitTrace(c, traceInfos)
		}
		if err != nil {
			e.EmitError(c, err)
		}
		// 如果已commit 表示返回数据已设置，无需处理
		if c.Committed {
			return
		}
		c.Committed = true
		if err != nil {
			e.Error(c, err)
		} else {
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
		}
	})
	return e
}

// GET add http get method handle
func (e *Elton) GET(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodGet, path, handlerList...)
}

// POST add http post method handle
func (e *Elton) POST(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodPost, path, handlerList...)
}

// PUT add http put method handle
func (e *Elton) PUT(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodPut, path, handlerList...)
}

// PATCH add http patch method handle
func (e *Elton) PATCH(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodPatch, path, handlerList...)
}

// DELETE add http delete method handle
func (e *Elton) DELETE(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodDelete, path, handlerList...)
}

// HEAD add http head method handle
func (e *Elton) HEAD(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodHead, path, handlerList...)
}

// OPTIONS add http options method handle
func (e *Elton) OPTIONS(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodOptions, path, handlerList...)
}

// TRACE add http trace method handle
func (e *Elton) TRACE(path string, handlerList ...Handler) *Elton {
	return e.Handle(http.MethodTrace, path, handlerList...)
}

// ALL add http all method handle
func (e *Elton) ALL(path string, handlerList ...Handler) *Elton {
	for _, method := range methods {
		e.Handle(method, path, handlerList...)
	}
	return e
}

// Use add middleware function handle
func (e *Elton) Use(handlerList ...Handler) *Elton {
	if e.Middlewares == nil {
		e.Middlewares = make([]Handler, 0)
	}
	for _, fn := range handlerList {
		name := e.GetFunctionName(fn)
		e.SetFunctionName(fn, name)
	}
	e.Middlewares = append(e.Middlewares, handlerList...)
	return e
}

// UseWithName add middleware and set function's name
func (e *Elton) UseWithName(handler Handler, name string) *Elton {
	e.SetFunctionName(handler, name)
	return e.Use(handler)
}

// Pre add pre middleware function handler
func (e *Elton) Pre(handlerList ...PreHandler) *Elton {
	if e.PreMiddlewares == nil {
		e.PreMiddlewares = make([]PreHandler, 0)
	}
	e.PreMiddlewares = append(e.PreMiddlewares, handlerList...)
	return e
}

// NotFound not found handle
func (e *Elton) NotFound(resp http.ResponseWriter, req *http.Request) *Elton {
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

// MethodNotAllowed method not allowed handle
func (e *Elton) MethodNotAllowed(resp http.ResponseWriter, req *http.Request) *Elton {
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

// Error error handle
func (e *Elton) Error(c *Context, err error) *Elton {
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

// EmitError emit error function
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

// OnError on error function
func (e *Elton) OnError(ln ErrorListener) *Elton {
	if e.errorListeners == nil {
		e.errorListeners = make([]ErrorListener, 0)
	}
	e.errorListeners = append(e.errorListeners, ln)
	return e
}

// EmitTrace emit trace
func (e *Elton) EmitTrace(c *Context, infos TraceInfos) *Elton {
	lns := e.traceListeners
	for _, ln := range lns {
		ln(c, infos)
	}
	return e
}

// OnTrace on trace function
func (e *Elton) OnTrace(ln TraceListener) *Elton {
	if e.traceListeners == nil {
		e.traceListeners = make([]TraceListener, 0)
	}
	e.traceListeners = append(e.traceListeners, ln)
	return e
}

// AddGroup add the group to elton
func (e *Elton) AddGroup(g *Group) *Elton {
	for _, r := range g.routers {
		e.Handle(r.Method, r.Path, r.HandleList...)
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

// GET add group http get method handler
func (g *Group) GET(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodGet, p, fns...)
}

// POST add group http post method handler
func (g *Group) POST(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodPost, p, fns...)
}

// PUT add group http put method handler
func (g *Group) PUT(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodPut, p, fns...)
}

// PATCH add group http patch method handler
func (g *Group) PATCH(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodPatch, p, fns...)
}

// DELETE add group http delete method handler
func (g *Group) DELETE(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodDelete, p, fns...)
}

// HEAD add group http head method handler
func (g *Group) HEAD(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodHead, p, fns...)
}

// OPTIONS add group http options method handler
func (g *Group) OPTIONS(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodOptions, p, fns...)
}

// TRACE add group http trace method handler
func (g *Group) TRACE(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	g.add(http.MethodTrace, p, fns...)
}

// ALL add group http all method handler
func (g *Group) ALL(path string, handlerList ...Handler) {
	p := g.Path + path
	fns := g.merge(handlerList)
	for _, method := range methods {
		g.add(method, p, fns...)
	}
}

// Compose compose handler list
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

func getMs(ns int) string {
	microSecond := int(time.Microsecond)
	milliSecond := int(time.Millisecond)
	if ns < microSecond {
		return "0"
	}

	// 计算ms的位
	ms := ns / milliSecond
	prefix := strconv.Itoa(ms)

	// 计算micro seconds
	offset := (ns % milliSecond) / microSecond
	// 如果小于10，不展示小数点（取小数点两位）
	unit := 10
	if offset < unit {
		return prefix
	}
	// 如果小于100，补一位0
	if offset < 100 {
		return prefix + ".0" + strconv.Itoa(offset/unit)
	}
	return prefix + "." + strconv.Itoa(offset/unit)
}

// ServerTiming trace infos to server timing
func (traceInfos TraceInfos) ServerTiming(prefix string) string {
	size := len(traceInfos)
	if size == 0 {
		return ""
	}

	// 转换为 http server timing
	s := new(strings.Builder)
	// 每一个server timing长度预估为30
	s.Grow(30 * size)
	for i, traceInfo := range traceInfos {
		v := traceInfo.Duration.Nanoseconds()
		s.WriteString(prefix)
		s.WriteString(strconv.Itoa(i))
		s.Write(ServerTimingDur)
		s.WriteString(getMs(int(v)))
		s.Write(ServerTimingDesc)
		s.WriteString(traceInfo.Name)
		s.Write(ServerTimingEnd)
		if i != size-1 {
			s.WriteRune(',')
		}

	}
	return s.String()
}
