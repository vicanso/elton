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
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReset(t *testing.T) {
	assert := assert.New(t)
	params := new(RouteParams)
	params.Add("key", "value")
	c := Context{
		Request:   httptest.NewRequest("GET", "https://aslant.site/", nil),
		Response:  httptest.NewRecorder(),
		Committed: true,
		ID:        "abcd",
		Route:     "/users/me",
		Next: func() error {
			return nil
		},
		Params:      params,
		StatusCode:  200,
		Body:        make(map[string]string),
		BodyBuffer:  bytes.NewBufferString("abcd"),
		RequestBody: []byte("abcd"),
		m:           make(map[interface{}]interface{}),
		realIP:      "abcd",
		clientIP:    "abcd",
		reuseStatus: ReuseContextEnabled,
	}
	c.Reset()
	assert.Nil(c.Request)
	assert.Nil(c.Response)
	assert.False(c.Committed)
	assert.Equal("", c.ID)
	assert.Equal("", c.Route)
	assert.Nil(c.Next)
	assert.Empty(params.Keys)
	assert.Empty(params.Values)
	assert.Equal(0, c.StatusCode)
	assert.Nil(c.Body)
	assert.Nil(c.BodyBuffer)
	assert.Nil(c.RequestBody)
	assert.Nil(c.m)
	assert.Equal("", c.realIP)
	assert.Equal("", c.clientIP)
	assert.Equal(int32(ReuseContextEnabled), c.reuseStatus)
}

func TestContext(t *testing.T) {
	data := "abcd"
	assert := assert.New(t)
	c := NewContext(nil, nil)
	assert.NotNil(c.Params)
	c.WriteHeader(http.StatusBadRequest)
	assert.Equal(c.StatusCode, http.StatusBadRequest)
	_, err := c.Write([]byte(data))
	assert.Nil(err)
	assert.Equal(data, c.BodyBuffer.String())
}

func TestRemoteAddr(t *testing.T) {
	assert := assert.New(t)
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	req.RemoteAddr = "192.168.1.1:7000"

	c := Context{
		Request: req,
	}
	assert.Equal("192.168.1.1", c.RemoteAddr())
}

func TestRealIP(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		newContext func() *Context
		ip         string
	}{
		// get from cache
		{
			newContext: func() *Context {
				req := httptest.NewRequest("GET", "/", nil)
				c := NewContext(nil, req)
				c.realIP = "abc"
				return c

			},
			ip: "abc",
		},
		// get from x-forwarded-for
		{
			newContext: func() *Context {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set(HeaderXForwardedFor, "192.0.0.1, 192.168.1.1")
				c := NewContext(nil, req)
				return c

			},
			ip: "192.0.0.1",
		},
		// get from x-real-ip
		{
			newContext: func() *Context {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set(HeaderXRealIP, "192.168.0.1")
				c := NewContext(nil, req)
				return c
			},
			ip: "192.168.0.1",
		},
		// get real ip from remote addr
		{
			newContext: func() *Context {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "192.168.1.1:7000"
				c := NewContext(nil, req)
				return c
			},
			ip: "192.168.1.1",
		},
	}
	for _, tt := range tests {
		c := tt.newContext()
		assert.Equal(tt.ip, c.RealIP())
	}
}

func TestGetClientIP(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		newContext func() *Context
		ip         string
	}{
		// get from cache
		{
			newContext: func() *Context {
				req := httptest.NewRequest("GET", "/", nil)
				c := NewContext(nil, req)
				c.clientIP = "abc"
				return c

			},
			ip: "abc",
		},
		// get from x-forwarded-for
		{
			newContext: func() *Context {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set(HeaderXForwardedFor, "192.168.1.1, 1.1.1.1, 2.2.2.2")
				c := NewContext(nil, req)
				return c

			},
			ip: "2.2.2.2",
		},
		// get from x-real-ip
		{
			newContext: func() *Context {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set(HeaderXRealIP, "1.1.1.1")
				c := NewContext(nil, req)
				return c
			},
			ip: "1.1.1.1",
		},
		// get by remote addr
		{
			newContext: func() *Context {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "192.168.1.1:7000"
				c := NewContext(nil, req)
				return c
			},
			ip: "192.168.1.1",
		},
	}
	for _, tt := range tests {
		c := tt.newContext()
		assert.Equal(tt.ip, c.ClientIP())
	}
}

func TestParam(t *testing.T) {
	assert := assert.New(t)
	c := Context{}
	assert.Equal(c.Param("name"), "", "params is not initialized, it should be nil")
	params := new(RouteParams)
	params.Add("name", "tree.xie")
	c.Params = params
	assert.Equal("tree.xie", c.Param("name"))
}

func TestQueryParam(t *testing.T) {
	assert := assert.New(t)
	req := httptest.NewRequest("GET", "https://aslant.site/?name=tree.xie", nil)
	resp := httptest.NewRecorder()
	c := NewContext(resp, req)
	assert.Equal("tree.xie", c.QueryParam("name"))
	assert.Empty(c.QueryParam("account"))
}

func TestQuery(t *testing.T) {
	assert := assert.New(t)
	req := httptest.NewRequest("GET", "https://aslant.site/?name=tree.xie&type=1", nil)
	c := NewContext(nil, req)
	q := c.Query()
	assert.Equal("tree.xie", q["name"])
	assert.Equal("1", q["type"])
}

func TestSetGet(t *testing.T) {
	assert := assert.New(t)
	c := Context{}
	value, _ := c.Get("name")
	assert.Nil(value, "should return nil when store is not initialized")
	c.Set("name", "tree.xie")
	value, _ = c.Get("name")
	assert.Equal("tree.xie", value.(string))

	i := 1
	c.Set("int", i)
	assert.Equal(i, c.GetInt("int"))

	var i64 int64 = 1
	c.Set("int64", i64)
	assert.Equal(i64, c.GetInt64("int64"))

	s := "s"
	c.Set("string", s)
	assert.Equal(s, c.GetString("string"))

	b := true
	c.Set("bool", b)
	assert.Equal(b, c.GetBool("bool"))

	var f32 float32 = 1.0
	c.Set("float32", f32)
	assert.Equal(f32, c.GetFloat32("float32"))

	f64 := 1.0
	c.Set("float64", f64)
	assert.Equal(f64, c.GetFloat64("float64"))

	now := time.Now()
	c.Set("time", now)
	assert.Equal(now, c.GetTime("time"))

	d := time.Second
	c.Set("duration", d)
	assert.Equal(d, c.GetDuration("duration"))

	arr := []string{
		"a",
	}
	c.Set("stringSlice", arr)
	assert.Equal(arr, c.GetStringSlice("stringSlice"))
}

func TestGetSetHeader(t *testing.T) {
	newContext := func() *Context {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Token", "abc")
		resp := httptest.NewRecorder()
		c := NewContext(resp, req)
		return c
	}

	t.Run("get header from request", func(t *testing.T) {
		c := newContext()
		assert := assert.New(t)
		assert.Equal("abc", c.GetRequestHeader("X-Token"))
	})

	t.Run("set header to request", func(t *testing.T) {
		c := newContext()
		key := "X-Request-ID"
		value := "1"
		assert := assert.New(t)
		assert.Equal(c.GetRequestHeader(key), "", "request id should be nil before set")
		c.SetRequestHeader(key, value)
		assert.Equal(value, c.GetRequestHeader(key))
		c.SetRequestHeader(key, "")
		assert.Empty(c.GetRequestHeader(key))
	})

	t.Run("add header to request", func(t *testing.T) {
		c := newContext()
		assert := assert.New(t)
		key := "X-Request-Type"
		c.AddRequestHeader(key, "1")
		c.AddRequestHeader(key, "2")
		ids := c.Request.Header[key]
		assert.Equal("1,2", strings.Join(ids, ","))
	})

	t.Run("set header to the response", func(t *testing.T) {
		c := newContext()
		assert := assert.New(t)
		c.SetHeader("X-Response-Id", "1")
		assert.Equal("1", c.GetHeader("X-Response-Id"))
	})

	t.Run("get header from response", func(t *testing.T) {
		c := newContext()
		assert := assert.New(t)
		idc := "GZ"
		key := "X-IDC"
		c.SetHeader(key, idc)
		assert.Equal(idc, c.GetHeader(key))
	})

	t.Run("get header of response", func(t *testing.T) {
		c := newContext()
		assert := assert.New(t)
		assert.NotNil(c.Header(), "response header should not be nil")
	})

	t.Run("reset header", func(t *testing.T) {
		c := newContext()
		c.SetHeader("a", "1")
		assert := assert.New(t)
		c.ResetHeader()
		assert.Equal(0, len(c.Header()))
	})
}

func TestGetKeys(t *testing.T) {
	assert := assert.New(t)
	c := NewContext(nil, nil)
	assert.Nil(c.getKeys())
	e := New()
	keys := []string{
		"a",
		"b",
	}
	ssk := &SimpleSignedKeys{
		keys: keys,
	}
	e.SignedKeys = ssk
	c.elton = e
	assert.Equal(keys, c.getKeys())
}

func TestCookie(t *testing.T) {
	newContext := func() *Context {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "a",
			Value: "b",
		})
		resp := httptest.NewRecorder()
		c := NewContext(resp, req)
		return c
	}

	t.Run("get cookie", func(t *testing.T) {
		assert := assert.New(t)
		c := newContext()
		cookie, err := c.Cookie("a")
		assert.Nil(err, "get cookie should be successful")
		assert.Equal("a", cookie.Name)
		assert.Equal("b", cookie.Value)
	})

	t.Run("set cookie", func(t *testing.T) {
		assert := assert.New(t)
		c := newContext()
		cookie := &http.Cookie{
			Name:     "a",
			Value:    "b",
			MaxAge:   300,
			Secure:   true,
			Path:     "/",
			HttpOnly: true,
		}
		c.AddCookie(cookie)
		assert.Equal("a=b; Path=/; Max-Age=300; HttpOnly; Secure", c.GetHeader(HeaderSetCookie))
	})

}

func TestSignedCookie(t *testing.T) {
	sk := new(RWMutexSignedKeys)
	sk.SetKeys([]string{
		"secret",
	})
	elton := &Elton{
		SignedKeys: sk,
	}
	t.Run("set signed cookie", func(t *testing.T) {
		assert := assert.New(t)
		resp := httptest.NewRecorder()
		c := NewContext(resp, nil)
		c.elton = elton
		cookie := &http.Cookie{
			Name:     "a",
			Value:    "b",
			MaxAge:   300,
			Secure:   true,
			Path:     "/",
			HttpOnly: true,
		}
		c.AddSignedCookie(cookie)
		assert.Equal("a=b; Path=/; Max-Age=300; HttpOnly; Secure,a.sig=jK8pWDfgnIdsDF73KVgdXnXvk63BBCDOcaqwVjasY-0; Path=/; Max-Age=300; HttpOnly; Secure", strings.Join(c.Header()[HeaderSetCookie], ","))
	})

	t.Run("get signed cookie", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "https://aslant.site/?name=tree.xie&type=1", nil)
		req.AddCookie(&http.Cookie{
			Name:  "a",
			Value: "b",
		})
		req.AddCookie(&http.Cookie{
			Name:  "a.sig",
			Value: "jK8pWDfgnIdsDF73KVgdXnXvk63BBCDOcaqwVjasY-0",
		})
		resp := httptest.NewRecorder()
		c := NewContext(resp, req)
		_, err := c.SignedCookie("a")
		assert.Equal(errSignKeyIsNil, err)

		c.elton = elton
		cookie, err := c.SignedCookie("a")
		assert.Nil(err, "signed cookie should be successful")
		assert.Equal("b", cookie.Value)
	})

	t.Run("get signed cookie(verify fail)", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "https://aslant.site/?name=tree.xie&type=1", nil)
		req.AddCookie(&http.Cookie{
			Name:  "a",
			Value: "b",
		})
		req.AddCookie(&http.Cookie{
			Name:  "a.sig",
			Value: "abcd",
		})
		resp := httptest.NewRecorder()
		c := NewContext(resp, req)
		c.elton = elton
		cookie, err := c.SignedCookie("a")
		assert.Equal(http.ErrNoCookie, err)
		assert.Nil(cookie)
	})

}

func TestSendFile(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		newContext func() *Context
		file       string
		err        error
	}{
		{
			newContext: func() *Context {
				c := NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
				return c
			},
			file: "abc.html",
			err:  ErrFileNotFound,
		},
		{
			newContext: func() *Context {
				c := NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
				return c
			},
			file: "docs/book.json",
		},
	}
	for _, tt := range tests {
		c := tt.newContext()
		err := c.SendFile(tt.file)
		assert.Equal(tt.err, err)
		if err == nil {
			assert.NotEmpty(c.GetHeader(HeaderContentLength))
			assert.NotEmpty(c.GetHeader(HeaderLastModified))
			assert.NotEmpty(c.GetHeader(HeaderContentType))
			assert.NotEmpty(c.Body)
		}
	}
}

func TestReadFile(t *testing.T) {
	assert := assert.New(t)

	testData := []byte("test data")
	fileName := "test.txt"
	// 生成http文件上传数据
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", fileName)
	if err != nil {
		return
	}
	_, err = io.Copy(fw, bytes.NewReader(testData))
	assert.Nil(err)
	err = w.Close()
	assert.Nil(err)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(b.Bytes()))
	req.Header = http.Header{
		"Content-Type": []string{
			w.FormDataContentType(),
		},
	}
	c := NewContext(httptest.NewRecorder(), req)
	data, fileHeader, err := c.ReadFile("file")
	assert.Nil(err)
	assert.Equal(fileName, fileHeader.Filename)
	assert.Equal(int64(9), fileHeader.Size)
	assert.Equal(testData, data)
}

func TestHTML(t *testing.T) {
	assert := assert.New(t)

	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	html := "<html><body></body></html>"
	c.HTML(html)
	assert.Equal("text/html; charset=utf-8", resp.Header().Get(HeaderContentType))
	assert.Equal(html, c.BodyBuffer.String())
}

func TestRedirect(t *testing.T) {
	assert := assert.New(t)
	req := httptest.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	c := NewContext(resp, req)
	err := c.Redirect(299, "")
	assert.Equal(err, ErrInvalidRedirect)

	url := "https://aslant.site/"
	err = c.Redirect(302, url)
	assert.Nil(err)
	assert.Equal(url, c.GetHeader(HeaderLocation))
	assert.Equal(302, c.StatusCode)
}

func TestCreate(t *testing.T) {
	assert := assert.New(t)
	body := "abc"
	c := NewContext(nil, nil)
	c.Created(body)
	assert.Equal(http.StatusCreated, c.StatusCode)
	assert.Equal(body, c.Body.(string))
}

func TestNoContent(t *testing.T) {
	assert := assert.New(t)
	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	c.SetHeader(HeaderContentType, "a")
	c.SetHeader(HeaderContentLength, "b")
	c.SetHeader(HeaderTransferEncoding, "c")
	c.NoContent()
	assert.Equal(http.StatusNoContent, c.StatusCode)
	assert.Nil(c.Body)
	assert.Nil(c.BodyBuffer)
	assert.Empty(c.GetHeader(HeaderContentType))
	assert.Empty(c.GetHeader(HeaderContentLength))
	assert.Empty(c.GetHeader(HeaderTransferEncoding))
}

func TestMergeHeader(t *testing.T) {
	assert := assert.New(t)
	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	h := make(http.Header)
	h.Add("a", "1")
	h.Add("a", "2")
	c.MergeHeader(h)
	assert.Equal(h, c.Header())
}

func TestNotModified(t *testing.T) {
	assert := assert.New(t)
	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	c.Body = map[string]string{}
	c.BodyBuffer = bytes.NewBufferString("abc")
	c.SetHeader(HeaderContentEncoding, "gzip")
	c.SetHeader(HeaderContentType, "text/html")
	c.NotModified()
	assert.Equal(http.StatusNotModified, c.StatusCode)
	assert.Nil(c.Body)
	assert.Nil(c.BodyBuffer)
	assert.Empty(c.GetHeader(HeaderContentEncoding))
	assert.Empty(c.GetHeader(HeaderContentType))
}

func TestCacheControl(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		newContext   func() *Context
		cacheControl string
	}{
		// no cache
		{
			newContext: func() *Context {
				resp := httptest.NewRecorder()
				c := NewContext(resp, nil)
				c.NoCache()
				return c
			},
			cacheControl: "no-cache",
		},
		// no store
		{
			newContext: func() *Context {
				resp := httptest.NewRecorder()
				c := NewContext(resp, nil)
				c.NoStore()
				return c
			},
			cacheControl: "no-store",
		},
		// max-age
		{
			newContext: func() *Context {
				resp := httptest.NewRecorder()
				c := NewContext(resp, nil)
				c.CacheMaxAge(time.Minute)
				return c
			},
			cacheControl: "public, max-age=60",
		},
		// s-maxage
		{
			newContext: func() *Context {
				resp := httptest.NewRecorder()
				c := NewContext(resp, nil)
				c.CacheMaxAge(time.Minute, 10*time.Second)
				return c
			},
			cacheControl: "public, max-age=60, s-maxage=10",
		},
		// private max-age
		{
			newContext: func() *Context {
				resp := httptest.NewRecorder()
				c := NewContext(resp, nil)
				c.PrivateCacheMaxAge(time.Minute)
				return c
			},
			cacheControl: "private, max-age=60",
		},
	}

	for _, tt := range tests {
		c := tt.newContext()
		assert.Equal(tt.cacheControl, c.GetHeader("Cache-Control"))
	}
}

func TestSetContentTypeByExt(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		newContext  func() *Context
		contentType string
	}{
		{
			newContext: func() *Context {
				resp := httptest.NewRecorder()
				c := NewContext(resp, nil)
				c.SetContentTypeByExt(".html")
				return c

			},
			contentType: "text/html; charset=utf-8",
		},
		{
			newContext: func() *Context {
				resp := httptest.NewRecorder()
				c := NewContext(resp, nil)
				c.SetContentTypeByExt("index.html")
				return c

			},
			contentType: "text/html; charset=utf-8",
		},
		{
			newContext: func() *Context {
				resp := httptest.NewRecorder()
				c := NewContext(resp, nil)
				c.SetContentTypeByExt("../abcd/index.html")
				return c

			},
			contentType: "text/html; charset=utf-8",
		},
	}

	for _, tt := range tests {
		c := tt.newContext()
		assert.Equal(tt.contentType, c.GetHeader(HeaderContentType))
	}
}

func TestDisableReuse(t *testing.T) {
	assert := assert.New(t)
	c := &Context{}
	assert.True(c.isReuse())
	c.DisableReuse()
	assert.False(c.isReuse())
}

func TestPush(t *testing.T) {
	assert := assert.New(t)
	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	err := c.Push("/a.css", nil)
	assert.Equal(ErrNotSupportPush, err)
}

func TestGetCod(t *testing.T) {
	assert := assert.New(t)
	c := NewContext(nil, nil)
	c.elton = &Elton{}
	assert.Equal(c.elton, c.Elton())
}
func TestNewContext(t *testing.T) {
	assert := assert.New(t)
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	resp := httptest.NewRecorder()
	c := NewContext(resp, req)
	assert.Equal(req, c.Request)
	assert.Equal(resp, c.Response)
}

func TestContextPass(t *testing.T) {
	assert := assert.New(t)
	e := New()
	another := New()
	another.GET("/", func(c *Context) error {
		c.BodyBuffer = bytes.NewBufferString("new data")
		return nil
	})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	resp := httptest.NewRecorder()
	e.GET("/", func(c *Context) error {
		c.Pass(another)
		// the data will be ignored
		c.BodyBuffer = bytes.NewBufferString("original data")
		return nil
	})
	e.ServeHTTP(resp, req)
	assert.Equal(http.StatusOK, resp.Code)
	assert.Equal("new data", resp.Body.String())
}

func TestContextServerTiming(t *testing.T) {
	assert := assert.New(t)
	traceInfos := make(TraceInfos, 0)
	traceInfos = append(traceInfos, &TraceInfo{
		Name:     "a",
		Duration: time.Microsecond * 10,
	})
	traceInfos = append(traceInfos, &TraceInfo{
		Name:     "b",
		Duration: time.Millisecond + time.Microsecond,
	})
	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	c.ServerTiming(traceInfos, "elton-")
	assert.Equal(`elton-0;dur=0.01;desc="a",elton-1;dur=1;desc="b"`, c.GetHeader(HeaderServerTiming))
}

func TestPipe(t *testing.T) {
	assert := assert.New(t)
	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	data := []byte("abcd")
	r := bytes.NewReader(data)
	written, err := c.Pipe(r)
	assert.Nil(err)
	assert.Equal(int64(len(data)), written)
	assert.Equal(data, resp.Body.Bytes())
}
