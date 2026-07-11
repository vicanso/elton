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
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vicanso/hes"
	"github.com/vicanso/keygrip"
)

// Status is the running status of elton
type Status int32

const (
	// StatusRunning running status
	StatusRunning Status = iota
	// StatusClosing closing status
	StatusClosing
	// StatusClosed closed status
	StatusClosed
)

// ErrServerNotInitialized the http server of elton is not initialized
var ErrServerNotInitialized = errors.New("server is not initialized")

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
		status atomic.Int32
		// route tree
		tree *node
		// routers all router infos
		routers []RouterInfo
		// middlewares middleware function
		middlewares []Handler
		// preMiddlewares pre middleware function
		preMiddlewares []PreHandler
		errorListeners []ErrorListener
		traceListeners []TraceListener
		// doneListeners request done
		doneListeners []DoneListener
		// beforeListeners before request handle
		beforeListeners []BeforeListener
		// functionInfos the function address:name map
		functionInfos map[uintptr]string
		// functionInfosMutex protects functionInfos for concurrent access
		functionInfosMutex sync.RWMutex
		// keygrip 缓存：避免每次 SignedCookie 都 keygrip.New
		kgMu    sync.Mutex
		kgKeys  []string
		kg      *keygrip.Keygrip
		ctxPool sync.Pool
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
		children    []*Group
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
	// DoneListener request done listener
	DoneListener func(*Context)
	// BeforeListener before request handle listener
	BeforeListener func(*Context)
	// PreHandler pre handler
	PreHandler func(*http.Request)
)

var _ http.Handler = (*Elton)(nil)
var _ context.Context = (*Context)(nil)

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
	e.ctxPool.New = func() any {
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

// IsIntranet reports whether s is a loopback, private, or link-local address.
// Invalid or empty values return false.
func IsIntranet(s string) bool {
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// SetFunctionName sets the name of handler function,
// it will use to http timing
func (e *Elton) SetFunctionName(fn any, name string) {
	p := reflect.ValueOf(fn).Pointer()
	e.functionInfosMutex.Lock()
	e.functionInfos[p] = name
	e.functionInfosMutex.Unlock()
}

// GetFunctionName return the name of handler function
func (e *Elton) GetFunctionName(fn any) string {
	p := reflect.ValueOf(fn).Pointer()
	e.functionInfosMutex.RLock()
	name := e.functionInfos[p]
	e.functionInfosMutex.RUnlock()
	if name != "" {
		return name
	}
	return runtime.FuncForPC(p).Name()
}

// ensureFunctionName returns the handler name, caching runtime name if unset.
func (e *Elton) ensureFunctionName(fn any) string {
	p := reflect.ValueOf(fn).Pointer()
	e.functionInfosMutex.RLock()
	name := e.functionInfos[p]
	e.functionInfosMutex.RUnlock()
	if name != "" {
		return name
	}
	name = runtime.FuncForPC(p).Name()
	e.functionInfosMutex.Lock()
	// double-check：并发 ensure 时只写一次
	if existing := e.functionInfos[p]; existing != "" {
		name = existing
	} else {
		e.functionInfos[p] = name
	}
	e.functionInfosMutex.Unlock()
	return name
}

// functionNameLocked resolves name; caller must hold functionInfosMutex (RLock).
func (e *Elton) functionNameLocked(fn Handler) string {
	p := reflect.ValueOf(fn).Pointer()
	if name := e.functionInfos[p]; name != "" {
		return name
	}
	return runtime.FuncForPC(p).Name()
}

// keygrip returns a cached keygrip for SignedKeys, rebuilding when keys change.
func (e *Elton) keygrip() *keygrip.Keygrip {
	if e.SignedKeys == nil {
		return nil
	}
	keys := e.SignedKeys.Keys()
	if len(keys) == 0 {
		return nil
	}
	e.kgMu.Lock()
	defer e.kgMu.Unlock()
	if e.kg != nil && slices.Equal(e.kgKeys, keys) {
		return e.kg
	}
	e.kgKeys = slices.Clone(keys)
	e.kg = keygrip.New(keys)
	return e.kg
}

// ListenAndServe listens the addr and serve http,
// it returns ErrServerNotInitialized if the server of elton is nil.
func (e *Elton) ListenAndServe(addr string) error {
	if e.Server == nil {
		return ErrServerNotInitialized
	}
	e.Server.Addr = addr
	return e.Server.ListenAndServe()
}

// ListenAndServeTLS listens the addr and serve https,
// it returns ErrServerNotInitialized if the server of elton is nil.
func (e *Elton) ListenAndServeTLS(addr, certFile, keyFile string) error {
	if e.Server == nil {
		return ErrServerNotInitialized
	}
	e.Server.Addr = addr
	return e.Server.ListenAndServeTLS(certFile, keyFile)
}

// Serve serves http server,
// it returns ErrServerNotInitialized if the server of elton is nil.
func (e *Elton) Serve(l net.Listener) error {
	if e.Server == nil {
		return ErrServerNotInitialized
	}
	return e.Server.Serve(l)
}

// Close closes the http server
func (e *Elton) Close() error {
	if e.Server == nil {
		return ErrServerNotInitialized
	}
	return e.Server.Close()
}

// Shutdown gracefully shuts down the http server without
// interrupting any active connections
func (e *Elton) Shutdown(ctx context.Context) error {
	if e.Server == nil {
		return ErrServerNotInitialized
	}
	return e.Server.Shutdown(ctx)
}

// GracefulClose closes the http server gracefully.
// It sets the status to be closing (rejecting new requests with 503),
// waits for the delay, then shuts down the server.
// ctx取消时停止等待并立即进入shutdown（此时Shutdown会关闭监听
// 并立即返回ctx.Err，不再等待活跃连接处理完成）。
func (e *Elton) GracefulClose(ctx context.Context, delay time.Duration) error {
	e.status.Store(int32(StatusClosing))
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
		case <-timer.C:
		}
	}
	e.status.Store(int32(StatusClosed))
	return e.Shutdown(ctx)
}

// Status returns status of elton
func (e *Elton) Status() Status {
	return Status(e.status.Load())
}

// Closing judge the status whether is closing
func (e *Elton) Closing() bool {
	return e.Status() == StatusClosing
}

// Running judge the status whether is running
func (e *Elton) Running() bool {
	return e.Status() == StatusRunning
}

// ServeHTTP http handler
func (e *Elton) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	status := e.Status()
	// 非运行中的状态
	if status != StatusRunning {
		resp.WriteHeader(http.StatusServiceUnavailable)
		_, err := fmt.Fprintf(resp, "service is not available, status is %d", status)
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

// Routers returns routers of elton
func (e *Elton) Routers() []RouterInfo {
	return slices.Clone(e.routers)
}

// Handle adds http handle function.
// 注册时将全局中间件（Use添加）与该路由的handler列表合并为一条执行链，
// 请求到达时通过c.Next()逐个推进（洋葱模型）：
//   - 任一环节返回error则中断，触发error监听器并输出错误响应；
//   - c.Committed为true时Next短路，不再执行后续handler；
//   - 链执行完成后，统一将BodyBuffer（或reader类型的Body）写出至响应。
//
// 注意：路由注册（Handle/GET/POST/Use等）应在服务启动前完成，
// 路由树不支持与请求处理并发修改。
func (e *Elton) Handle(method, path string, handlerList ...Handler) *Elton {
	for _, fn := range handlerList {
		e.ensureFunctionName(fn)
	}

	e.routers = append(e.routers, RouterInfo{
		Method: method,
		Route:  path,
	})
	e.tree.InsertRoute(methodTypeMap[method], path, func(c *Context) {
		if e.beforeListeners != nil {
			e.emitBefore(c)
		}
		if e.doneListeners != nil {
			defer e.emitDone(c)
		}
		c.Route = path
		mids := e.middlewares
		maxMid := len(mids)
		maxNext := maxMid + len(handlerList)
		index := -1
		var trace *Trace
		// handlerNames 在 EnableTrace 时一次性解析，避免每层 Next 加锁查名
		var handlerNames []string
		if e.EnableTrace {
			trace = &Trace{
				Infos: make(TraceInfos, 0, maxNext),
			}
			c.WithContext(context.WithValue(c.Context(), ContextTraceKey, trace))
			handlerNames = make([]string, maxNext)
			e.functionInfosMutex.RLock()
			for i := 0; i < maxMid; i++ {
				handlerNames[i] = e.functionNameLocked(mids[i])
			}
			for i, fn := range handlerList {
				handlerNames[maxMid+i] = e.functionNameLocked(fn)
			}
			e.functionInfosMutex.RUnlock()
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
			fnName := handlerNames[index]
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
			// 出错时reader body不会被输出，关闭避免资源泄漏
			c.closeReaderBody()
			e.error(c, err)
			return
		}
		// 需要在设置status code之前设置响应长度
		if c.BodyBuffer != nil {
			// BodyBuffer优先输出，若Body为未使用的reader则关闭
			c.closeReaderBody()
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
	return e.Multi(methods, path, handlerList...)
}

// Multi adds multi method
func (e *Elton) Multi(methods []string, path string, handlerList ...Handler) *Elton {
	for _, method := range methods {
		e.Handle(method, path, handlerList...)
	}
	return e
}

// Use adds middleware handler function to elton's middleware list
func (e *Elton) Use(handlerList ...Handler) *Elton {
	for _, fn := range handlerList {
		e.ensureFunctionName(fn)
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
	status := http.StatusInternalServerError
	message := err.Error()
	he := &hes.Error{}
	if errors.As(err, &he) {
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
		elton:    e,
	}, err)
}

// OnError adds listen to error event
func (e *Elton) OnError(ln ErrorListener) *Elton {
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
	e.traceListeners = append(e.traceListeners, ln)
	return e
}

// OnDone adds listen to request done, it will be triggered
// when the request handle is done
func (e *Elton) OnDone(ln DoneListener) *Elton {
	e.doneListeners = append(e.doneListeners, ln)
	return e
}

func (e *Elton) emitDone(c *Context) {
	for _, ln := range e.doneListeners {
		ln(c)
	}
}

// OnBefore adds listen to before request done(after pre middlewares, before middlewares)
func (e *Elton) OnBefore(ln BeforeListener) *Elton {
	e.beforeListeners = append(e.beforeListeners, ln)
	return e
}

func (e *Elton) emitBefore(c *Context) {
	for _, ln := range e.beforeListeners {
		ln(c)
	}
}

// AddGroup adds the group and its sub groups to elton
func (e *Elton) AddGroup(groups ...*Group) *Elton {
	for _, g := range groups {
		for _, r := range g.routers {
			e.Handle(r.Method, r.Path, r.HandleList...)
		}
		e.AddGroup(g.children...)
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

// NewGroup returns a new sub group of the group,
// the path and handler list will be merged with the parent's.
// The sub group will be added to elton together with its parent
// by elton.AddGroup.
func (g *Group) NewGroup(path string, handlerList ...Handler) *Group {
	child := &Group{
		Path:        g.Path + path,
		HandlerList: g.merge(handlerList),
	}
	g.children = append(g.children, child)
	return child
}

func (g *Group) handle(method, path string, handlerList ...Handler) *Group {
	g.routers = append(g.routers, &Router{
		Method:     method,
		Path:       g.Path + path,
		HandleList: g.merge(handlerList),
	})
	return g
}

// GET adds http get method handler to group
func (g *Group) GET(path string, handlerList ...Handler) *Group {
	return g.handle(http.MethodGet, path, handlerList...)
}

// POST adds http post method handler to group
func (g *Group) POST(path string, handlerList ...Handler) *Group {
	return g.handle(http.MethodPost, path, handlerList...)
}

// PUT adds http put method handler to group
func (g *Group) PUT(path string, handlerList ...Handler) *Group {
	return g.handle(http.MethodPut, path, handlerList...)
}

// PATCH adds http patch method handler to group
func (g *Group) PATCH(path string, handlerList ...Handler) *Group {
	return g.handle(http.MethodPatch, path, handlerList...)
}

// DELETE adds http delete method handler to group
func (g *Group) DELETE(path string, handlerList ...Handler) *Group {
	return g.handle(http.MethodDelete, path, handlerList...)
}

// HEAD adds http head method handler to group
func (g *Group) HEAD(path string, handlerList ...Handler) *Group {
	return g.handle(http.MethodHead, path, handlerList...)
}

// OPTIONS adds http options method handler to group
func (g *Group) OPTIONS(path string, handlerList ...Handler) *Group {
	return g.handle(http.MethodOptions, path, handlerList...)
}

// TRACE adds http trace method handler to group
func (g *Group) TRACE(path string, handlerList ...Handler) *Group {
	return g.handle(http.MethodTrace, path, handlerList...)
}

// ALL adds http all methods handler to group
func (g *Group) ALL(path string, handlerList ...Handler) *Group {
	return g.Multi(methods, path, handlerList...)
}

// Multi adds multi http methods handler to group
func (g *Group) Multi(methods []string, path string, handlerList ...Handler) *Group {
	for _, method := range methods {
		g.handle(method, path, handlerList...)
	}
	return g
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
