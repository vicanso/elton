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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/hes"
)

func TestIntranet(t *testing.T) {
	assert := assert.New(t)

	assert.True(IsIntranet("127.0.0.1"))
	assert.True(IsIntranet("192.168.1.1"))
	assert.False(IsIntranet("1.1.1.1"))
}

func TestSkipper(t *testing.T) {
	c := &Context{
		Committed: true,
	}
	assert := assert.New(t)
	assert.Equal(true, DefaultSkipper(c), "default skip should return true")

	e := New()
	execFisrtMid := false
	execSecondMid := false
	e.Use(func(c *Context) error {
		execFisrtMid = true
		c.Committed = true
		return c.Next()
	})
	e.Use(func(c *Context) error {
		execSecondMid = true
		return c.Next()
	})

	e.GET("/", func(c *Context) error {
		return nil
	})
	req := httptest.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)
	assert.Equal(true, execFisrtMid)
	assert.Equal(false, execSecondMid)
}

func TestListenAndServe(t *testing.T) {
	assert := assert.New(t)
	e := New()
	go func() {
		_ = e.ListenAndServe("")
	}()
	time.Sleep(10 * time.Millisecond)
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)
	assert.Equal(resp.Code, http.StatusNotFound)
	err := e.Close()
	assert.Nil(err, "close server should be successful")
}

func TestServe(t *testing.T) {
	assert := assert.New(t)
	e := New()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(err, "net listen should be successful")
	go func() {
		_ = e.Serve(ln)
	}()
	time.Sleep(10 * time.Millisecond)
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)
	assert.Equal(http.StatusNotFound, resp.Code)
	err = e.Close()
	assert.Nil(err, "close server should be successful")
}

func TestNewWithoutServer(t *testing.T) {
	e := NewWithoutServer()
	assert := assert.New(t)
	assert.Nil(e.Server, "new without server should be nil")
}

func TestPreHandle(t *testing.T) {
	e := New()
	pong := "pong"
	e.GET("/ping", func(c *Context) error {
		c.BodyBuffer = bytes.NewBufferString(pong)
		return nil
	})
	t.Run("not found", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "/api/ping", nil)
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(404, resp.Code)
		assert.Equal("Not Found", resp.Body.String())
	})
	t.Run("method not allow", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("POST", "/ping", nil)
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(405, resp.Code)
		assert.Equal("Method Not Allowed", resp.Body.String())
	})

	t.Run("pong", func(t *testing.T) {
		// replace url prefix /api
		urlPrefix := "/api"
		e.Pre(func(req *http.Request) {
			path := req.URL.Path
			if strings.HasPrefix(path, urlPrefix) {
				req.URL.Path = path[len(urlPrefix):]
			}
		})

		assert := assert.New(t)
		req := httptest.NewRequest("GET", urlPrefix+"/ping", nil)
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(200, resp.Code)
		assert.Equal(pong, resp.Body.String())
	})
}

func TestHandle(t *testing.T) {
	e := New()
	t.Run("all methods", func(t *testing.T) {
		assert := assert.New(t)
		path := "/test-path"
		e.GET(path)
		e.POST(path)
		e.PUT(path)
		e.PATCH(path)
		e.DELETE(path)
		e.HEAD(path)
		e.OPTIONS(path)
		e.TRACE(path)
		allMethods := "/all-methods"
		e.ALL(allMethods)
		for index, r := range e.GetRouters() {
			p := path
			if index >= len(methods) {
				p = allMethods
			}
			assert.Equal(p, r.Route)
		}
		assert.Equal(2*len(methods), len(e.GetRouters()), "method handle add fail")
	})
	t.Run("group", func(t *testing.T) {
		assert := assert.New(t)
		key := "count"
		countValue := 4
		fn := func(c *Context) error {
			c.Set(key, 1)
			return c.Next()
		}
		e.UseWithName(fn, "test")
		e.Use(func(c *Context) error {
			v := c.GetInt(key)
			c.Set(key, v+1)
			return c.Next()
		}, func(c *Context) error {
			v := c.GetInt(key)
			c.Set(key, v+2)
			return c.Next()
		})
		userGroupPath := "/users"
		userGroup := NewGroup(userGroupPath, func(c *Context) error {
			assert.Equal(true, strings.HasPrefix(c.Request.URL.Path, userGroupPath), "group route should have the same url prefix")
			return c.Next()
		})
		doneCount := 0
		userGroup.ALL("/me", func(c *Context) (err error) {
			v := c.GetInt(key)
			assert.Equal(countValue, v)
			assert.Equal(userGroupPath+"/me", c.Route, "route url is invalid")
			doneCount++
			return
		})
		e.AddGroup(userGroup)
		for _, method := range methods {
			req := httptest.NewRequest(method, "https://aslant.site/users/me", nil)
			resp := httptest.NewRecorder()
			e.ServeHTTP(resp, req)
		}
		assert.Equal("test", e.GetFunctionName(fn))
		assert.Equal(len(methods), doneCount, "not all method request is done")
	})

	route := "/system/info"

	t.Run("test method handler", func(t *testing.T) {
		assert := assert.New(t)

		for _, method := range []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"HEAD",
			"OPTIONS",
			"TRACE",
		} {
			done := false
			sysGroup := NewGroup("/system")
			fn := sysGroup.GET
			switch method {
			case "GET":
				fn = sysGroup.GET
			case "POST":
				fn = sysGroup.POST
			case "PUT":
				fn = sysGroup.PUT
			case "PATCH":
				fn = sysGroup.PATCH
			case "DELETE":
				fn = sysGroup.DELETE
			case "HEAD":
				fn = sysGroup.HEAD
			case "OPTIONS":
				fn = sysGroup.OPTIONS
			case "TRACE":
				fn = sysGroup.TRACE
			}
			fn("/info", func(c *Context) (err error) {
				c.StatusCode = 201
				c.BodyBuffer = bytes.NewBufferString("abcd")
				assert.Equal(route, c.Route)
				done = true
				return
			})
			e.AddGroup(sysGroup)
			req := httptest.NewRequest(method, "https://aslant.site/system/info", nil)
			resp := httptest.NewRecorder()
			e.ServeHTTP(resp, req)
			assert.Equal(true, done, "route handler isn't called")
			assert.Equal(201, resp.Code)
		}
	})

	t.Run("params", func(t *testing.T) {
		assert := assert.New(t)
		e.GET("/params/:id", func(c *Context) error {
			assert.Equal("1", c.Param("id"), "get route param fail")
			return nil
		})
		req := httptest.NewRequest("GET", "https://aslant.site/params/1", nil)
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
	})

	t.Run("not found", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "https://aslant.site/not-found", nil)
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(http.StatusNotFound, resp.Code)
		assert.Equal("Not Found", resp.Body.String())
	})

	t.Run("error", func(t *testing.T) {
		assert := assert.New(t)
		customErr := hes.New("abcd")
		e.GET("/error", func(c *Context) error {
			return customErr
		})
		req := httptest.NewRequest("GET", "https://aslant.site/error", nil)
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(http.StatusBadRequest, resp.Code, "default hes error status code should be 400")
		assert.Equal("statusCode=400, message=abcd", resp.Body.String())
	})

	t.Run("get routers", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal(34, len(e.GetRouters()), "router count fail")
	})

	t.Run("response body reader", func(t *testing.T) {
		assert := assert.New(t)
		data := "abcd"
		e.GET("/index.html", func(c *Context) error {
			c.Body = bytes.NewReader([]byte(data))
			return nil
		})
		req := httptest.NewRequest("GET", "https://aslant.site/index.html", nil)
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(http.StatusOK, resp.Code)
		assert.Equal(data, resp.Body.String())
	})
}

func TestErrorHandler(t *testing.T) {
	t.Run("remove header", func(t *testing.T) {
		assert := assert.New(t)
		e := New()
		resp := httptest.NewRecorder()
		c := NewContext(resp, nil)
		keys := []string{
			HeaderETag,
			HeaderLastModified,
			HeaderContentEncoding,
			HeaderContentLength,
		}
		for _, key := range keys {
			c.SetHeader(key, "a")
		}
		message := "abcd"
		e.error(c, errors.New(message))
		for _, key := range keys {
			value := c.GetHeader(key)
			assert.Equal(value, "", "the "+key+" header should be nil")
		}
		assert.Equal(http.StatusInternalServerError, resp.Code, "default error status should be 500")
		assert.Equal(message, resp.Body.String())
	})

	t.Run("custom error handler", func(t *testing.T) {
		assert := assert.New(t)
		e := New()
		e.GET("/", func(c *Context) error {
			return hes.New("abc")
		})

		done := false
		e.ErrorHandler = func(c *Context, err error) {
			done = true
		}
		req := httptest.NewRequest("GET", "/", nil)
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(true, done, "custom error handler should be called")
	})
}

func TestNotFoundHandler(t *testing.T) {
	assert := assert.New(t)
	e := New()
	e.GET("/", func(c *Context) error {
		return nil
	})
	done := false
	e.NotFoundHandler = func(resp http.ResponseWriter, req *http.Request) {
		done = true
	}
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)
	assert.Equal(true, done, "custom not found handler should be called")
}

func TestOnError(t *testing.T) {
	assert := assert.New(t)
	e := New()
	c := NewContext(nil, nil)
	customErr := hes.New("abc")
	e.EmitError(c, customErr)
	e.OnError(func(_ *Context, err error) {
		assert.Equal(customErr, err)
	})
	e.EmitError(c, customErr)
	req, err := http.NewRequest("GET", "/", nil)
	assert.Nil(err)

	e.emitError(httptest.NewRecorder(), req, customErr)
}

func TestOnTrace(t *testing.T) {
	assert := assert.New(t)
	e := New()
	e.EnableTrace = true
	done := false
	e.OnTrace(func(c *Context, infos TraceInfos) {
		assert.Equal(len(infos), 2, "trace count should be 2")
		done = true
	})
	e.Use(func(c *Context) error {
		return c.Next()
	})
	ignoreFn := func(c *Context) error {
		return c.Next()
	}
	e.Use(ignoreFn)
	e.SetFunctionName(ignoreFn, "-")

	e.GET("/users/me", func(c *Context) error {
		return nil
	})
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)
	assert.Equal(true, done, "on trace should be called")
}

func TestOnBefore(t *testing.T) {
	assert := assert.New(t)
	e := New()

	onBefore := false
	e.OnBefore(func(ctx *Context) {
		onBefore = true
	})
	e.GET("/", func(ctx *Context) error {
		assert.True(onBefore)

		return nil
	})
	req := httptest.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)
	assert.True(onBefore)
}

func TestOnDone(t *testing.T) {
	assert := assert.New(t)
	e := New()

	done := false
	e.OnDone(func(ctx *Context) {
		done = true
	})
	e.GET("/", func(ctx *Context) error {
		return nil
	})
	req := httptest.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)
	assert.True(done)
}

func TestGenerateID(t *testing.T) {
	assert := assert.New(t)
	e := New()
	randID := "abc"
	e.GenerateID = func() string {
		return randID
	}
	e.GET("/", func(c *Context) error {
		assert.Equal(randID, c.ID, "generate id function should be called")
		return nil
	})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)
}

func TestCompose(t *testing.T) {
	t.Run("run success", func(t *testing.T) {
		assert := assert.New(t)
		e := New()
		index := 0
		fn1 := func(c *Context) (err error) {
			assert.Equal(0, index)
			index++
			err = c.Next()
			index++
			assert.Equal(6, index)
			return
		}
		fn2 := func(c *Context) (err error) {
			assert.Equal(1, index)
			index++
			err = c.Next()
			index++
			assert.Equal(5, index)
			return
		}
		fn3 := func(c *Context) (err error) {
			assert.Equal(2, index)
			index++
			err = c.Next()
			index++
			assert.Equal(4, index)
			return
		}
		fn := Compose(fn1, fn2, fn3)
		e.Use(fn)
		e.Use(func(c *Context) (err error) {
			assert.Equal(3, index)
			return c.Next()
		})
		e.GET("/", func(c *Context) (err error) {
			assert.Equal(3, index)
			c.BodyBuffer = bytes.NewBufferString("abcd")
			return
		})
		req := httptest.NewRequest("GET", "https://aslant.site/", nil)
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(200, resp.Code)
		assert.Equal("abcd", resp.Body.String())
	})

	t.Run("error", func(t *testing.T) {
		assert := assert.New(t)
		e := New()
		fn := Compose(func(c *Context) error {
			return c.Next()
		}, func(c *Context) error {
			return errors.New("custom error")
		})
		e.Use(fn)
		e.GET("/", func(c *Context) (err error) {
			c.BodyBuffer = bytes.NewBufferString("abcd")
			return
		})
		req := httptest.NewRequest("GET", "https://aslant.site/", nil)
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(500, resp.Code)
		assert.Equal("custom error", resp.Body.String())
	})
}
func TestGetSetFunctionName(t *testing.T) {
	assert := assert.New(t)
	fn := func() {}
	e := New()
	fnName := "test"
	e.SetFunctionName(fn, fnName)
	assert.Equal(fnName, e.GetFunctionName(fn))
}

func TestContextWithContext(t *testing.T) {
	assert := assert.New(t)
	req := httptest.NewRequest("GET", "/", nil)
	c := NewContext(nil, req)

	assert.Equal(req.Context(), c.Context())

	ctx, cancel := context.WithTimeout(c.Context(), time.Second)
	defer cancel()
	c.WithContext(ctx)

	assert.Equal(ctx, c.Context())
}

func TestGracefulClose(t *testing.T) {
	e := New()
	t.Run("running 404", func(t *testing.T) {
		assert := assert.New(t)
		resp := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/users/me", nil)
		e.ServeHTTP(resp, req)
		assert.Equal(http.StatusNotFound, resp.Code)
	})

	t.Run("graceful close", func(t *testing.T) {
		assert := assert.New(t)
		done := make(chan bool)
		go func() {
			err := e.GracefulClose(time.Second)
			assert.Nil(err, "server close should be successful")
			done <- true
		}()
		time.Sleep(10 * time.Millisecond)
		assert.Equal(e.GetStatus(), int32(StatusClosing), "server status should be closing")
		assert.True(e.Closing())
		assert.False(e.Running())
		resp := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/users/me", nil)
		e.ServeHTTP(resp, req)
		assert.Equal(http.StatusServiceUnavailable, resp.Code)
		assert.Equal("service is not available, status is 1", resp.Body.String())

		<-done
		assert.Equal(int32(StatusClosed), e.GetStatus(), "server status should be closed")
	})
}

// https://stackoverflow.com/questions/50120427/fail-unit-tests-if-coverage-is-below-certain-percentage
func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	rc := m.Run()

	// rc 0 means we've passed,
	// and CoverMode will be non empty if run with -cover
	if rc == 0 && testing.CoverMode() != "" {
		c := testing.Coverage()
		if c < 0.9 {
			fmt.Println("Tests passed but coverage failed at", c)
			rc = -1
		}
	}
	os.Exit(rc)
}
