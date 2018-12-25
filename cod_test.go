package cod

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListenAndServe(t *testing.T) {
	d := New()
	go d.ListenAndServe("")
	err := d.Close()
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
		userGroup := d.Group(userGroupPath, func(c *Context) error {
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
		for _, method := range methods {
			req := httptest.NewRequest(method, "https://aslant.site/users/me", nil)
			resp := httptest.NewRecorder()
			d.ServeHTTP(resp, req)
		}
		if doneCount != len(methods) {
			t.Fatalf("route handle fail")
		}
	})

	sysGroup := d.Group("/system")
	route := "/system/info"
	t.Run("get", func(t *testing.T) {
		done := false
		sysGroup.GET("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		req := httptest.NewRequest("GET", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("post", func(t *testing.T) {
		done := false
		sysGroup.POST("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		req := httptest.NewRequest("POST", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("put", func(t *testing.T) {
		done := false
		sysGroup.PUT("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		req := httptest.NewRequest("PUT", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("patch", func(t *testing.T) {
		done := false
		sysGroup.PATCH("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		req := httptest.NewRequest("PATCH", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("delete", func(t *testing.T) {
		done := false
		sysGroup.DELETE("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		req := httptest.NewRequest("DELETE", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("head", func(t *testing.T) {
		done := false
		sysGroup.HEAD("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		req := httptest.NewRequest("HEAD", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("options", func(t *testing.T) {
		done := false
		sysGroup.OPTIONS("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
		req := httptest.NewRequest("OPTIONS", "https://aslant.site/system/info", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("handle function is not call")
		}
	})

	t.Run("trace", func(t *testing.T) {
		done := false
		sysGroup.TRACE("/info", func(c *Context) (err error) {
			if c.Route != route {
				t.Fatalf("route param fail")
			}
			done = true
			return
		})
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
	etag := GenerateETag([]byte(""))
	if etag != `"0-2jmj7l5rSw0yVb_vlWAYkK_YBwk="` {
		t.Fatalf("gen nil byte etag fail")
	}
	etag = GenerateETag([]byte("abc"))
	if etag != `"3-qZk-NkcGgWq6PiVxeFDCbJzQ2J0="` {
		t.Fatalf("gen abc etag fail")
	}
}
