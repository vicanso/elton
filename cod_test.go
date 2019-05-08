package cod

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/hes"
)

func TestSkipper(t *testing.T) {
	c := &Context{
		Committed: true,
	}
	assert := assert.New(t)
	assert.Equal(DefaultSkipper(c), true)
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
	assert.Equal(resp.Code, http.StatusNotFound)
	err = d.Close()
	assert.Nil(err, "close server should be successful")
}

func TestNewWithoutServer(t *testing.T) {
	d := NewWithoutServer()
	assert := assert.New(t)
	assert.Nil(d.Server, "new without server should be nil")
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
			assert.Equal(r.Path, p)
		}
		assert.Equal(len(d.Routers), 16)
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
			assert.Equal(strings.HasPrefix(c.Request.URL.Path, userGroupPath), true)
			return c.Next()
		})
		doneCount := 0
		userGroup.ALL("/me", func(c *Context) (err error) {
			v := c.Get(key).(int)
			assert.Equal(v, countValue)
			assert.Equal(c.Route, "/users/me")
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
				assert.Equal(c.Route, route)
				done = true
				return
			})
			d.AddGroup(sysGroup)
			req := httptest.NewRequest(method, "https://aslant.site/system/info", nil)
			resp := httptest.NewRecorder()
			d.ServeHTTP(resp, req)
			assert.Equal(done, true)
			assert.Equal(resp.Code, 201)
		}
	})

	t.Run("params", func(t *testing.T) {
		assert := assert.New(t)
		d.GET("/params/:id", func(c *Context) error {
			assert.Equal(c.Param("id"), "1")
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
		assert.Equal(resp.Code, http.StatusNotFound)
		assert.Equal(resp.Body.String(), "Not found")
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
		assert.Equal(resp.Code, http.StatusBadRequest)
		assert.Equal(resp.Body.String(), "message=abcd")
	})

	t.Run("get routers", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal(len(d.Routers), 34, "router count fail")
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
	assert.Equal(resp.Code, 500)
	assert.Equal(resp.Body.String(), "id should be 5 numbers")
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
		d.Error(c, errors.New("abcd"))
		for _, key := range keys {
			value := c.GetHeader(key)
			assert.Equal(value, "", "the "+key+" header should be nil")
		}
		assert.Equal(resp.Code, http.StatusInternalServerError)
		assert.Equal(resp.Body.String(), "abcd")
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
		assert.Equal(done, true, "custom error handler should be called")
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
	assert.Equal(done, true, "custom not found handler should be called")
}

func TestOnError(t *testing.T) {
	assert := assert.New(t)
	d := New()
	c := NewContext(nil, nil)
	customErr := hes.New("abc")
	d.EmitError(c, customErr)
	d.OnError(func(_ *Context, err error) {
		assert.Equal(err, customErr)
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
	assert.Equal(done, true, "on trace should be called")
}

func TestGenerateID(t *testing.T) {
	assert := assert.New(t)
	d := New()
	randID := "abc"
	d.GenerateID = func() string {
		return randID
	}
	d.GET("/", func(c *Context) error {
		assert.Equal(c.ID, randID)
		return nil
	})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
}

func TestGetSetFunctionName(t *testing.T) {
	assert := assert.New(t)
	fn := func() {}
	d := New()
	fnName := "test"
	d.SetFunctionName(fn, "test")
	assert.Equal(d.GetFunctionName(fn), fnName)
}

func TestConvertToServerTiming(t *testing.T) {
	assert := assert.New(t)
	traceInfos := make(TraceInfos, 0)

	t.Run("get ms", func(t *testing.T) {
		assert.Equal(getMs(10), "0")
		assert.Equal(getMs(100000), "0.10")
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
		assert.Equal(string(traceInfos.ServerTiming("cod-")), `cod-0;dur=0.01;desc="a",cod-1;dur=1;desc="b"`)
	})
}

func TestGracefulClose(t *testing.T) {
	d := New()
	t.Run("running 404", func(t *testing.T) {
		assert := assert.New(t)
		resp := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/users/me", nil)
		d.ServeHTTP(resp, req)
		assert.Equal(resp.Code, http.StatusNotFound)
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
		assert.Equal(resp.Code, http.StatusServiceUnavailable)
		assert.Equal(resp.Body.String(), "service is not available, status is 1")

		<-done
		assert.Equal(d.GetStatus(), int32(StatusClosed), "server status should be closed")
	})
}
