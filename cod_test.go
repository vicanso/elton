package cod

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/hes"
)

func TestIsPrivateIP(t *testing.T) {
	assert := assert.New(t)
	assert.True(IsPrivateIP(net.ParseIP("127.0.0.1")))
	assert.True(IsPrivateIP(net.ParseIP("10.0.0.1")))
	assert.True(IsPrivateIP(net.ParseIP("172.16.0.1")))
}

func TestSkipper(t *testing.T) {
	c := &Context{
		Committed: true,
	}
	assert := assert.New(t)
	assert.Equal(true, DefaultSkipper(c), "default skip should return true")

	d := New()
	execFisrtMid := false
	execSecondMid := false
	d.Use(func(c *Context) error {
		execFisrtMid = true
		c.Committed = true
		return c.Next()
	})
	d.Use(func(c *Context) error {
		execSecondMid = true
		return c.Next()
	})

	d.GET("/", func(c *Context) error {
		return nil
	})
	req := httptest.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	assert.Equal(true, execFisrtMid)
	assert.Equal(false, execSecondMid)
}

func TestListenAndServe(t *testing.T) {
	assert := assert.New(t)
	d := New()
	go d.ListenAndServe("")
	time.Sleep(10 * time.Millisecond)
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	assert.Equal(resp.Code, http.StatusNotFound)
	err := d.Close()
	assert.Nil(err, "close server should be successful")
}

func TestServe(t *testing.T) {
	assert := assert.New(t)
	d := New()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(err, "net listen should be successful")
	go d.Serve(ln)
	time.Sleep(10 * time.Millisecond)
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	assert.Equal(http.StatusNotFound, resp.Code)
	err = d.Close()
	assert.Nil(err, "close server should be successful")
}

func TestNewWithoutServer(t *testing.T) {
	d := NewWithoutServer()
	assert := assert.New(t)
	assert.Nil(d.Server, "new without server should be nil")
}

func TestPreHandle(t *testing.T) {
	d := New()
	pong := "pong"
	d.GET("/ping", func(c *Context) error {
		c.BodyBuffer = bytes.NewBufferString(pong)
		return nil
	})
	t.Run("not found", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "/api/ping", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		assert.Equal(404, resp.Code)
		assert.Equal("Not found", resp.Body.String())
	})

	t.Run("pong", func(t *testing.T) {
		// replace url prefix /api
		urlPrefix := "/api"
		d.Pre(func(req *http.Request) {
			path := req.URL.Path
			if strings.HasPrefix(path, urlPrefix) {
				req.URL.Path = path[len(urlPrefix):]
			}
		})

		assert := assert.New(t)
		req := httptest.NewRequest("GET", urlPrefix+"/ping", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		assert.Equal(200, resp.Code)
		assert.Equal(pong, resp.Body.String())
	})
}

func TestHandle(t *testing.T) {
	d := New()
	t.Run("all methods", func(t *testing.T) {
		assert := assert.New(t)
		path := "/test-path"
		d.GET(path)
		d.POST(path)
		d.PUT(path)
		d.PATCH(path)
		d.DELETE(path)
		d.HEAD(path)
		d.OPTIONS(path)
		d.TRACE(path)
		allMethods := "/all-methods"
		d.ALL(allMethods)
		for index, r := range d.Routers {
			p := path
			if index >= 8 {
				p = allMethods
			}
			assert.Equal(p, r.Path)
		}
		assert.Equal(16, len(d.Routers), "method handle add fail")
	})
	t.Run("group", func(t *testing.T) {
		assert := assert.New(t)
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
			assert.Equal(true, strings.HasPrefix(c.Request.URL.Path, userGroupPath), "group route should have the same url prefix")
			return c.Next()
		})
		doneCount := 0
		userGroup.ALL("/me", func(c *Context) (err error) {
			v := c.Get(key).(int)
			assert.Equal(countValue, v)
			assert.Equal(userGroupPath+"/me", c.Route, "route url is invalid")
			doneCount++
			return
		})
		d.AddGroup(userGroup)
		for _, method := range methods {
			req := httptest.NewRequest(method, "https://aslant.site/users/me", nil)
			resp := httptest.NewRecorder()
			d.ServeHTTP(resp, req)
		}
		assert.Equal(doneCount, len(methods), "not all method request is done")
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
			d.AddGroup(sysGroup)
			req := httptest.NewRequest(method, "https://aslant.site/system/info", nil)
			resp := httptest.NewRecorder()
			d.ServeHTTP(resp, req)
			assert.Equal(true, done, "route handler isn't called")
			assert.Equal(201, resp.Code)
		}
	})

	t.Run("params", func(t *testing.T) {
		assert := assert.New(t)
		d.GET("/params/:id", func(c *Context) error {
			assert.Equal("1", c.Param("id"), "get route param fail")
			return nil
		})
		req := httptest.NewRequest("GET", "https://aslant.site/params/1", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
	})

	t.Run("not found", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "https://aslant.site/not-found", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		assert.Equal(http.StatusNotFound, resp.Code)
		assert.Equal("Not found", resp.Body.String())
	})

	t.Run("error", func(t *testing.T) {
		assert := assert.New(t)
		customErr := hes.New("abcd")
		d.GET("/error", func(c *Context) error {
			return customErr
		})
		req := httptest.NewRequest("GET", "https://aslant.site/error", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		assert.Equal(http.StatusBadRequest, resp.Code, "default hes error status code should be 400")
		assert.Equal("message=abcd", resp.Body.String())
	})

	t.Run("get routers", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal(34, len(d.Routers), "router count fail")
	})

	t.Run("response body reader", func(t *testing.T) {
		assert := assert.New(t)
		data := "abcd"
		d.GET("/index.html", func(c *Context) error {
			c.Body = bytes.NewReader([]byte(data))
			return nil
		})
		req := httptest.NewRequest("GET", "https://aslant.site/index.html", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		assert.Equal(http.StatusOK, resp.Code)
		assert.Equal(data, resp.Body.String())
	})
}

func TestParamValidate(t *testing.T) {
	d := New()
	runMid := false
	assert := assert.New(t)
	d.AddValidator("id", func(value string) error {
		reg := regexp.MustCompile(`^[0-9]{5}$`)
		if !reg.MatchString(value) {
			return errors.New("id should be 5 numbers")
		}
		return nil
	})
	d.Use(func(c *Context) error {
		runMid = true
		return c.Next()
	})
	d.GET("/:id", func(c *Context) error {
		c.NoContent()
		return nil
	})
	req := httptest.NewRequest("GET", "/1", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	assert.True(runMid)
	assert.Equal(500, resp.Code)
	assert.Equal("id should be 5 numbers", resp.Body.String())
}

func TestErrorHandler(t *testing.T) {
	t.Run("remove header", func(t *testing.T) {
		assert := assert.New(t)
		d := New()
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
		d.Error(c, errors.New(message))
		for _, key := range keys {
			value := c.GetHeader(key)
			assert.Equal(value, "", "the "+key+" header should be nil")
		}
		assert.Equal(http.StatusInternalServerError, resp.Code, "default error status should be 500")
		assert.Equal(message, resp.Body.String())
	})

	t.Run("custom error handler", func(t *testing.T) {
		assert := assert.New(t)
		d := New()
		d.GET("/", func(c *Context) error {
			return hes.New("abc")
		})

		done := false
		d.ErrorHandler = func(c *Context, err error) {
			done = true
		}
		req := httptest.NewRequest("GET", "/", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		assert.Equal(true, done, "custom error handler should be called")
	})
}

func TestNotFoundHandler(t *testing.T) {
	assert := assert.New(t)
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
	assert.Equal(true, done, "custom not found handler should be called")
}

func TestOnError(t *testing.T) {
	assert := assert.New(t)
	d := New()
	c := NewContext(nil, nil)
	customErr := hes.New("abc")
	d.EmitError(c, customErr)
	d.OnError(func(_ *Context, err error) {
		assert.Equal(customErr, err)
	})
	d.EmitError(c, customErr)
}

func TestOnTrace(t *testing.T) {
	assert := assert.New(t)
	d := New()
	d.EnableTrace = true
	done := false
	d.OnTrace(func(c *Context, infos TraceInfos) {
		assert.Equal(len(infos), 2, "trace count should be 2")
		done = true
	})
	d.Use(func(c *Context) error {
		return c.Next()
	})
	ignoreFn := func(c *Context) error {
		return c.Next()
	}
	d.Use(ignoreFn)
	d.SetFunctionName(ignoreFn, "-")

	d.GET("/users/me", func(c *Context) error {
		return nil
	})
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	assert.Equal(true, done, "on trace should be called")
}

func TestGenerateID(t *testing.T) {
	assert := assert.New(t)
	d := New()
	randID := "abc"
	d.GenerateID = func() string {
		return randID
	}
	d.GET("/", func(c *Context) error {
		assert.Equal(randID, c.ID, "generate id function should be called")
		return nil
	})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
}

func TestCompose(t *testing.T) {
	t.Run("run success", func(t *testing.T) {
		assert := assert.New(t)
		d := New()
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
		d.Use(fn)
		d.Use(func(c *Context) (err error) {
			assert.Equal(3, index)
			return c.Next()
		})
		d.GET("/", func(c *Context) (err error) {
			assert.Equal(3, index)
			c.BodyBuffer = bytes.NewBufferString("abcd")
			return
		})
		req := httptest.NewRequest("GET", "https://aslant.site/", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		assert.Equal(200, resp.Code)
		assert.Equal("abcd", resp.Body.String())
	})

	t.Run("error", func(t *testing.T) {
		assert := assert.New(t)
		d := New()
		fn := Compose(func(c *Context) error {
			return c.Next()
		}, func(c *Context) error {
			return errors.New("custom error")
		})
		d.Use(fn)
		d.GET("/", func(c *Context) (err error) {
			c.BodyBuffer = bytes.NewBufferString("abcd")
			return
		})
		req := httptest.NewRequest("GET", "https://aslant.site/", nil)
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		assert.Equal(500, resp.Code)
		assert.Equal("custom error", resp.Body.String())
	})
}
func TestGetSetFunctionName(t *testing.T) {
	assert := assert.New(t)
	fn := func() {}
	d := New()
	fnName := "test"
	d.SetFunctionName(fn, fnName)
	assert.Equal(fnName, d.GetFunctionName(fn))
}

func TestConvertToServerTiming(t *testing.T) {
	assert := assert.New(t)
	traceInfos := make(TraceInfos, 0)

	t.Run("get ms", func(t *testing.T) {
		assert.Equal("0", getMs(10))
		assert.Equal("0.10", getMs(100000))
	})

	t.Run("empty trace infos", func(t *testing.T) {
		assert.Empty(traceInfos.ServerTiming(""), "no trace should return nil")
	})
	t.Run("server timing", func(t *testing.T) {
		traceInfos = append(traceInfos, &TraceInfo{
			Name:     "a",
			Duration: time.Microsecond * 10,
		})
		traceInfos = append(traceInfos, &TraceInfo{
			Name:     "b",
			Duration: time.Millisecond + time.Microsecond,
		})
		assert.Equal(`cod-0;dur=0.01;desc="a",cod-1;dur=1;desc="b"`, string(traceInfos.ServerTiming("cod-")))
	})
}

func TestGracefulClose(t *testing.T) {
	d := New()
	t.Run("running 404", func(t *testing.T) {
		assert := assert.New(t)
		resp := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/users/me", nil)
		d.ServeHTTP(resp, req)
		assert.Equal(http.StatusNotFound, resp.Code)
	})

	t.Run("graceful close", func(t *testing.T) {
		assert := assert.New(t)
		done := make(chan bool)
		go func() {
			err := d.GracefulClose(time.Second)
			assert.Nil(err, "server close should be successful")
			done <- true
		}()
		time.Sleep(10 * time.Millisecond)
		assert.Equal(d.GetStatus(), int32(StatusClosing), "server status should be closing")
		resp := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/users/me", nil)
		d.ServeHTTP(resp, req)
		assert.Equal(http.StatusServiceUnavailable, resp.Code)
		assert.Equal("service is not available, status is 1", resp.Body.String())

		<-done
		assert.Equal(int32(StatusClosed), d.GetStatus(), "server status should be closed")
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
