package cod

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type (
	// Cod web framework instance
	Cod struct {
		Server          *http.Server
		Router          *httprouter.Router
		Middlewares     []Handle
		errorLinsteners []ErrorLinstener
		ErrorHandle     ErrorHandle
		GenerateID      GenerateID
	}
	// Group group router
	Group struct {
		Path       string
		HandleList []Handle
		Cod        *Cod
	}
	// ErrorHandle error handle function
	ErrorHandle func(error, *Context)
	// GenerateID generate id
	GenerateID func() string
	// Handle cod handle function
	Handle func(*Context) error
	// ErrorLinstener error listener function
	ErrorLinstener func(*Context, error)
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
		if d.GenerateID != nil {
			c.ID = d.GenerateID()
		}
		c.Route = path
		c.cod = d
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
		if err != nil {
			d.EmitError(c, err)
			fn := d.ErrorHandle
			if fn == nil {
				fn = d.Error
			}
			fn(err, c)
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

// Use add middleware function handle
func (d *Cod) Use(handleList ...Handle) {
	d.Middlewares = append(d.Middlewares, handleList...)
}

// NotFound not found handle
func (d *Cod) NotFound(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusNotFound)
	resp.Write([]byte("Not found"))
}

// Error error handle
func (d *Cod) Error(err error, c *Context) {
	resp := c.Response
	he, ok := err.(*HTTPError)
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
	lns := d.errorLinsteners
	for _, ln := range lns {
		ln(c, err)
	}
}

// OnError on error function
func (d *Cod) OnError(ln ErrorLinstener) {
	if d.errorLinsteners == nil {
		d.errorLinsteners = make([]ErrorLinstener, 0)
	}
	d.errorLinsteners = append(d.errorLinsteners, ln)
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
	g.Cod.GET(p, fns...)
}

// POST add group http post method handl
func (g *Group) POST(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.POST(p, fns...)
}

// PUT add group http put method handl
func (g *Group) PUT(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.PUT(p, fns...)
}

// PATCH add group http patch method handl
func (g *Group) PATCH(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.PATCH(p, fns...)
}

// DELETE add group http delete method handl
func (g *Group) DELETE(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.DELETE(p, fns...)
}

// HEAD add group http head method handl
func (g *Group) HEAD(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.HEAD(p, fns...)
}

// OPTIONS add group http options method handl
func (g *Group) OPTIONS(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.OPTIONS(p, fns...)
}

// TRACE add group http trace method handl
func (g *Group) TRACE(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.TRACE(p, fns...)
}

// ALL add group http all method handl
func (g *Group) ALL(path string, handleList ...Handle) {
	p := g.Path + path
	fns := g.merge(handleList)
	g.Cod.ALL(p, fns...)
}
