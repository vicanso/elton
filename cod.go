package cod

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type (
	// Cod web framework instance
	Cod struct {
		Server      *http.Server
		Router      *httprouter.Router
		Middlewares []Handle
	}
	// Group group router
	Group struct {
		Path       string
		HandleList []Handle
		Cod        *Cod
	}
	// Handle cod handle function
	Handle func(c *Context) error
)

// New create a cod instance
func New() *Cod {
	d := &Cod{
		Router:      httprouter.New(),
		Middlewares: make([]Handle, 0),
	}
	s := &http.Server{
		Handler: d,
	}
	d.Server = s
	return d
}

// ListenAndServe listen and serve for http server
func (d *Cod) ListenAndServe(addr string) error {
	// TODO 调整可以指定Server的相关参数
	d.Server.Addr = addr
	return d.Server.ListenAndServe()
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

// Handle add http handle function
func (d *Cod) Handle(method, path string, handleList ...Handle) {
	d.Router.Handle(method, path, func(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
		c := NewContext(resp, req)
		defer ReleaseContext(c)
		c.Params = make(map[string]string)
		for _, item := range params {
			c.Params[item.Key] = item.Value
		}
		c.Route = path
		mids := d.Middlewares
		maxMid := len(mids)
		index := -1
		c.Next = func() error {
			index++
			// 如果已到最后，执行handle list
			if index >= maxMid {
				return handleList[index-maxMid](c)
			}
			return mids[index](c)
		}
		err := c.Next()
		// TODO 如果有未处理出错了 emit error
		if err != nil {
			d.Error(err, c)
		}
	})
}

// GET add http get method handle
func (d *Cod) GET(path string, handleList ...Handle) {
	d.Handle(http.MethodGet, path, handleList...)
}

// POST add http post method handle
func (d *Cod) POST(path string, handleList ...Handle) {
	d.Handle(http.MethodPost, path, handleList...)
}

// PUT add http put method handle
func (d *Cod) PUT(path string, handleList ...Handle) {
	d.Handle(http.MethodPut, path, handleList...)
}

// PATCH add http patch method handle
func (d *Cod) PATCH(path string, handleList ...Handle) {
	d.Handle(http.MethodPatch, path, handleList...)
}

// DELETE add http delete method handle
func (d *Cod) DELETE(path string, handleList ...Handle) {
	d.Handle(http.MethodDelete, path, handleList...)
}

// HEAD add http head method handle
func (d *Cod) HEAD(path string, handleList ...Handle) {
	d.Handle(http.MethodHead, path, handleList...)
}

// CONNECT add http connect method handle
func (d *Cod) CONNECT(path string, handleList ...Handle) {
	d.Handle(http.MethodConnect, path, handleList...)
}

// OPTIONS add http options method handle
func (d *Cod) OPTIONS(path string, handleList ...Handle) {
	d.Handle(http.MethodOptions, path, handleList...)
}

// TRACE add http trace method handle
func (d *Cod) TRACE(path string, handleList ...Handle) {
	d.Handle(http.MethodTrace, path, handleList...)
}

// ALL add http all method handle
func (d *Cod) ALL(path string, handleList ...Handle) {
	for _, method := range methods {
		d.Handle(method, path, handleList...)
	}
}

// Group create a http handle group
func (d *Cod) Group(path string, handleList ...Handle) (g *Group) {
	return &Group{
		Cod:        d,
		Path:       path,
		HandleList: handleList,
	}
}

// Add add middleware function handle
func (d *Cod) Add(handle Handle) {
	d.Middlewares = append(d.Middlewares, handle)
}

// NotFound not found handle
func (d *Cod) NotFound(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusNotFound)
	resp.Write([]byte("Not found"))
}

// Error error handle
func (d *Cod) Error(err error, c *Context) {
	// TODO error的处理优化
	resp := c.Response
	resp.WriteHeader(http.StatusInternalServerError)
	resp.Write([]byte(err.Error()))
}

func (g *Group) merge(s2 []Handle) []Handle {
	s1 := g.HandleList
	fns := make([]Handle, len(s1)+len(s2))
	copy(fns, s1)
	copy(fns[len(s1):], s2)
	return fns
}

// GET add group http get method handl
func (g *Group) GET(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.Handle(http.MethodGet, p, fns...)
}

// POST add group http post method handl
func (g *Group) POST(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.Handle(http.MethodPost, p, fns...)
}

// PATCH add group http patch method handl
func (g *Group) PATCH(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.Handle(http.MethodPatch, p, fns...)
}

// DELETE add group http delete method handl
func (g *Group) DELETE(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.Handle(http.MethodDelete, p, fns...)
}

// HEAD add group http head method handl
func (g *Group) HEAD(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.Handle(http.MethodHead, p, fns...)
}

// CONNECT add group http connect method handl
func (g *Group) CONNECT(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.Handle(http.MethodConnect, p, fns...)
}

// OPTIONS add group http options method handl
func (g *Group) OPTIONS(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.Handle(http.MethodOptions, p, fns...)
}

// TRACE add group http trace method handl
func (g *Group) TRACE(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.Handle(http.MethodTrace, p, fns...)
}

// ALL add group http all method handl
func (g *Group) ALL(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	for _, method := range methods {
		g.Cod.Handle(method, p, fns...)
	}
}
