package cod

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestListenAndServe(t *testing.T) {
	d := New()
	go d.ListenAndServe("")
	time.Sleep(10 * time.Millisecond)
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status code should be 404")
	}
	err := d.Close()
	if err != nil {
		t.Fatalf("close server fail, %v", err)
	}
}

func TestServe(t *testing.T) {
	d := New()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("serve fail, %v", err)
	}
	go d.Serve(ln)
	time.Sleep(10 * time.Millisecond)
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status code should be 404")
	}
	err = d.Close()
	if err != nil {
		t.Fatalf("close server fail, %v", err)
	}
}

func TestNewWithoutServer(t *testing.T) {
	d := NewWithoutServer()
	if d.Server != nil {
		t.Fatalf("new without server fail")
	}
}

func TestIngoreNext(t *testing.T) {
	d := New()
	pass := false

	d.Use(func(c *Context) error {
		pass = true
		c.IgnoreNext = true
		return c.Next()
	})

	d.Use(func(c *Context) error {
		pass = false
		return c.Next()
	})
	d.GET("/", func(c *Context) error {
		pass = false
		return nil
	})
	req := httptest.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	if !pass {
		t.Fatalf("ingore next fail")
	}
}

func TestHandle(t *testing.T) {
	d := New()
	t.Run("group", func(t *testing.T) {
		key := "count"
		countValue := 4
		d.Use(func(c *Context) error {
			c.Set(key, 1)
			return c.Next()
		})
		d.Use(func(c *Context) error {
			v := c.Get(key).(int)
			c.Set(key, v+1)
			return c.Next()
		}, func(c *Context) error {
			v := c.Get(key).(int)
			c.Set(key, v+2)
			return c.Next()
		})
		userGroupPath := "/users"
		userGroup := NewGroup(userGroupPath, func(c *Context) error {
			if !strings.HasPrefix(c.Request.URL.Path, userGroupPath) {
				t.Fatalf("group handle fail")
			}
			return c.Next()
		})
		doneCount := 0
		userGroup.ALL("/me", func(c *Context) (err error) {
			v := c.Get(key).(int)
			if v != countValue {
				t.Fatalf(c.Request.Method + " handle fail")
			}
			if c.Route != "/users/me" {
				t.Fatalf("handle route is not match")
			}
			doneCount++
			return
		})
		d.AddGroup(userGroup)
		for _, method := range methods {
			req := httptest.NewRequest(method, "https://aslant.site/users/me", nil)
			resp := httptest.NewRecorder()
			d.ServeHTTP(resp, req)
		}
		if doneCount != len(methods) {
			t.Fatalf("route handle fail")
		}
	})

	route := "/system/info"
	t.Run("get", func(t *testing.T) {
		done := false
		sysGroup := NewGroup("/system")
		sysGroup.GET("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		d.AddGroup(sysGroup)
		req := httptest.NewRequest("GET", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("post", func(t *testing.T) {
		done := false
		sysGroup := NewGroup("/system")
		sysGroup.POST("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		d.AddGroup(sysGroup)
		req := httptest.NewRequest("POST", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("put", func(t *testing.T) {
		done := false
		sysGroup := NewGroup("/system")
		sysGroup.PUT("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		d.AddGroup(sysGroup)
		req := httptest.NewRequest("PUT", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("patch", func(t *testing.T) {
		done := false
		sysGroup := NewGroup("/system")
		sysGroup.PATCH("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		d.AddGroup(sysGroup)
		req := httptest.NewRequest("PATCH", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("delete", func(t *testing.T) {
		done := false
		sysGroup := NewGroup("/system")
		sysGroup.DELETE("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		d.AddGroup(sysGroup)
		req := httptest.NewRequest("DELETE", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("head", func(t *testing.T) {
		done := false
		sysGroup := NewGroup("/system")
		sysGroup.HEAD("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		d.AddGroup(sysGroup)
		req := httptest.NewRequest("HEAD", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("options", func(t *testing.T) {
		done := false
		sysGroup := NewGroup("/system")
		sysGroup.OPTIONS("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		d.AddGroup(sysGroup)
		req := httptest.NewRequest("OPTIONS", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("trace", func(t *testing.T) {
		done := false
		sysGroup := NewGroup("/system")
		sysGroup.TRACE("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		d.AddGroup(sysGroup)
		req := httptest.NewRequest("TRACE", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("params", func(t *testing.T) {
		d.GET("/params/:id", func(c *Context) error {
			if c.Param("id") == "" {
				t.Fatalf("set params fail")
			}
			return nil
		})
		req := httptest.NewRequest("GET", "https://aslant.site/params/1", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://aslant.site/not-found", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound ||
			resp.Body.String() != "Not found" {
			t.Fatalf("default not found handle fail")
		}
	})

	t.Run("error", func(t *testing.T) {
		customErr := errors.New("abcd")
		d.GET("/error", func(c *Context) error {
			return customErr
		})
		req := httptest.NewRequest("GET", "https://aslant.site/error", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if resp.Code != http.StatusInternalServerError ||
			resp.Body.String() != "abcd" {
			t.Fatalf("default error handle fail")
		}
	})

	t.Run("get routers", func(t *testing.T) {
		if len(d.Routers) != 18 {
			t.Fatalf("get routers fail")
		}
	})
}

func TestErrorHandler(t *testing.T) {
	d := New()
	d.GET("/", func(c *Context) error {
		return errors.New("abc")
	})

	done := false
	d.ErrorHandler = func(c *Context, err error) {
		done = true
	}
	req := httptest.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	if !done {
		t.Fatalf("custom error handler is not called")
	}
}

func TestNotFoundHandler(t *testing.T) {
	d := New()
	d.GET("/", func(c *Context) error {
		return nil
	})
	done := false
	d.NotFoundHandler = func(resp http.ResponseWriter, req *http.Request) {
		done = true
	}
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	if !done {
		t.Fatalf("custom not found handler is not called")
	}
}

func TestOnError(t *testing.T) {
	d := New()
	c := NewContext(nil, nil)
	cutstomErr := errors.New("abc")
	d.EmitError(c, cutstomErr)
	d.OnError(func(_ *Context, err error) {
		if err != cutstomErr {
			t.Fatalf("on error fail")
		}
	})
	d.EmitError(c, cutstomErr)
}

func TestOnTrace(t *testing.T) {
	d := New()
	d.EnableTrace = true
	done := false
	d.OnTrace(func(c *Context, infos []*TraceInfo) {
		if len(infos) != 2 {
			t.Fatalf("trace count should be 2")
		}
		done = true
	})
	d.Use(func(c *Context) error {
		return c.Next()
	})
	d.GET("/users/me", func(c *Context) error {
		return nil
	})
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	if !done {
		t.Fatalf("trace fail")
	}

}

func TestGenerateID(t *testing.T) {
	d := New()
	randID := "abc"
	d.GenerateID = func() string {
		return randID
	}
	d.GET("/", func(c *Context) error {
		if c.ID != randID {
			t.Fatalf("generate id fail")
		}
		return nil
	})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
}

func TestGenerateETag(t *testing.T) {
	eTag := GenerateETag([]byte(""))
	if eTag != `"0-2jmj7l5rSw0yVb_vlWAYkK_YBwk="` {
		t.Fatalf("gen nil byte eTag fail")
	}
	eTag = GenerateETag([]byte("abc"))
	if eTag != `"3-qZk-NkcGgWq6PiVxeFDCbJzQ2J0="` {
		t.Fatalf("gen abc eTag fail")
	}
}

func TestGetSetFunctionName(t *testing.T) {
	fn := func() {}
	d := New()
	fnName := "test"
	d.SetFunctionName(fn, "test")
	if d.GetFunctionName(fn) != fnName {
		t.Fatalf("get function name fail")
	}
}
