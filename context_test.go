package elton

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReset(t *testing.T) {
	assert := assert.New(t)
	c := Context{
		Request:   httptest.NewRequest("GET", "https://aslant.site/", nil),
		Response:  httptest.NewRecorder(),
		Headers:   make(http.Header),
		Committed: true,
		ID:        "abcd",
		Route:     "/users/me",
		Next: func() error {
			return nil
		},
		Params:        make(map[string]string),
		StatusCode:    200,
		Body:          make(map[string]string),
		BodyBuffer:    bytes.NewBufferString("abcd"),
		RequestBody:   []byte("abcd"),
		m:             make(map[interface{}]interface{}),
		realIP:        "abcd",
		clientIP:      "abcd",
		elton:         &Elton{},
		reuseDisabled: true,
	}
	c.Reset()
	assert.Nil(c.Request)
	assert.Nil(c.Response)
	assert.Nil(c.Headers)
	assert.False(c.Committed)
	assert.Equal("", c.ID)
	assert.Equal("", c.Route)
	assert.Nil(c.Next)
	assert.Nil(c.Params)
	assert.Equal(0, c.StatusCode)
	assert.Nil(c.Body)
	assert.Nil(c.BodyBuffer)
	assert.Nil(c.RequestBody)
	assert.Nil(c.m)
	assert.Equal("", c.realIP)
	assert.Equal("", c.clientIP)
	assert.Nil(c.elton)
	assert.False(c.reuseDisabled)
}

func TestContext(t *testing.T) {
	data := "abcd"
	assert := assert.New(t)
	c := NewContext(nil, nil)
	c.WriteHeader(http.StatusBadRequest)
	assert.Equal(c.StatusCode, http.StatusBadRequest)
	c.Write([]byte(data))

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
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	req.RemoteAddr = "192.168.1.1:7000"

	c := Context{
		Request: req,
	}
	t.Run("get real ip from cache", func(t *testing.T) {
		assert := assert.New(t)
		ip := "abc"
		c.realIP = ip
		assert.Equal(ip, c.RealIP())
		c.realIP = ""
	})

	t.Run("get from x-forwarded-for", func(t *testing.T) {
		assert := assert.New(t)
		defer req.Header.Del(HeaderXForwardedFor)
		req.Header.Set(HeaderXForwardedFor, "192.0.0.1, 192.168.1.1")
		assert.Equal("192.0.0.1", c.RealIP(), "real ip should get from x-forwarded-for")
		c.realIP = ""
	})

	t.Run("get from x-real-ip", func(t *testing.T) {
		defer req.Header.Del(HeaderXRealIP)
		req.Header.Set(HeaderXRealIP, "192.168.0.1")
		assert := assert.New(t)
		assert.Equal("192.168.0.1", c.RealIP(), "real ip should get from x-real-ip")
		c.realIP = ""
	})

	t.Run("get real ip from remote addr", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal("192.168.1.1", c.RealIP())
		c.realIP = ""
	})
}

func TestGetClientIP(t *testing.T) {
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	req.RemoteAddr = "192.168.1.1:7000"

	c := Context{
		Request: req,
	}
	t.Run("get client ip from cache", func(t *testing.T) {
		assert := assert.New(t)
		ip := "abc"
		c.clientIP = ip
		assert.Equal(ip, c.ClientIP())
		c.clientIP = ""
	})

	t.Run("get from x-forwarded-for", func(t *testing.T) {
		assert := assert.New(t)
		defer req.Header.Del(HeaderXForwardedFor)
		req.Header.Set(HeaderXForwardedFor, "192.168.1.1, 1.1.1.1, 2.2.2.2")
		assert.Equal("1.1.1.1", c.ClientIP(), "client ip shold get the first public ip from x-forwarded-for")
		c.clientIP = ""
	})

	t.Run("get from x-real-ip", func(t *testing.T) {
		assert := assert.New(t)
		defer req.Header.Del(HeaderXRealIP)
		req.Header.Set(HeaderXRealIP, "192.168.1.2")
		// real ip的是内网IP，因此取remote addr
		assert.Equal("192.168.1.1", c.ClientIP())

		c.clientIP = ""
		req.Header.Set(HeaderXRealIP, "1.1.1.1")
		assert.Equal("1.1.1.1", c.ClientIP())
	})
}

func TestParam(t *testing.T) {
	assert := assert.New(t)
	c := Context{}
	assert.Equal(c.Param("name"), "", "params is not initialized, it should be nil")
	c.Params = map[string]string{
		"name": "tree.xie",
	}
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

	req = httptest.NewRequest("GET", "https://aslant.site/", nil)
	c = NewContext(nil, req)
	assert.Nil(c.Query())
}

func TestSetGet(t *testing.T) {
	assert := assert.New(t)
	c := Context{}
	assert.Nil(c.Get("name"), "should return nil when store is not initialized")
	c.Set("name", "tree.xie")
	assert.Equal("tree.xie", c.Get("name").(string))
}

func TestGetSetHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "https://aslant.site/?name=tree.xie&type=1", nil)
	req.Header.Set("X-Token", "abc")
	resp := httptest.NewRecorder()
	c := NewContext(resp, req)

	t.Run("get header from request", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal("abc", c.GetRequestHeader("X-Token"))
	})

	t.Run("set header to request", func(t *testing.T) {
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
		assert := assert.New(t)
		key := "X-Request-Type"
		c.AddRequestHeader(key, "1")
		c.AddRequestHeader(key, "2")
		ids := c.Request.Header[key]
		assert.Equal("1,2", strings.Join(ids, ","))
	})

	t.Run("set header to the response", func(t *testing.T) {
		assert := assert.New(t)
		c.SetHeader("X-Response-Id", "1")
		assert.Equal("1", c.GetHeader("X-Response-Id"))
	})

	t.Run("get header from response", func(t *testing.T) {
		assert := assert.New(t)
		idc := "GZ"
		key := "X-IDC"
		c.SetHeader(key, idc)
		assert.Equal(idc, c.GetHeader(key))
	})

	t.Run("get header of response", func(t *testing.T) {
		assert := assert.New(t)
		assert.NotNil(c.Header(), "response header should not be nil")
	})

	t.Run("reset header", func(t *testing.T) {
		assert := assert.New(t)
		c.ResetHeader()
		assert.Equal(0, len(c.Header()))
	})
}

func TestGetKeys(t *testing.T) {
	assert := assert.New(t)
	c := NewContext(nil, nil)
	assert.Nil(c.getKeys())
	d := New()
	keys := []string{
		"a",
		"b",
	}
	ssk := &SimpleSignedKeys{
		keys: keys,
	}
	d.SignedKeys = ssk
	c.elton = d
	assert.Equal(keys, c.getKeys())
}

func TestCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "https://aslant.site/?name=tree.xie&type=1", nil)
	req.AddCookie(&http.Cookie{
		Name:  "a",
		Value: "b",
	})
	resp := httptest.NewRecorder()
	c := NewContext(resp, req)
	t.Run("get cookie", func(t *testing.T) {
		assert := assert.New(t)
		cookie, err := c.Cookie("a")
		assert.Nil(err, "get cookie should be successful")
		assert.Equal("a", cookie.Name)
		assert.Equal("b", cookie.Value)
	})

	t.Run("set cookie", func(t *testing.T) {
		assert := assert.New(t)
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
		assert.Equal("a=b; Path=/; Max-Age=300; HttpOnly; Secure,a.sig=9yv2rWFijew8K8a5Uw9jxRJE53s; Path=/; Max-Age=300; HttpOnly; Secure", strings.Join(c.Headers[HeaderSetCookie], ","))
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
			Value: "9yv2rWFijew8K8a5Uw9jxRJE53s",
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

func TestRedirect(t *testing.T) {
	assert := assert.New(t)
	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	err := c.Redirect(299, "")
	assert.Equal(err, ErrInvalidRedirect)

	url := "https://aslant.site/"
	err = c.Redirect(302, url)
	assert.Nil(err)
	assert.Equal(url, c.GetHeader(HeaderLocation))
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

func TestNotModified(t *testing.T) {
	assert := assert.New(t)
	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	c.Body = map[string]string{}
	c.BodyBuffer = bytes.NewBufferString("abc")
	c.Headers.Set(HeaderContentEncoding, "gzip")
	c.Headers.Set(HeaderContentType, "text/html")
	c.NotModified()
	assert.Equal(http.StatusNotModified, c.StatusCode)
	assert.Nil(c.Body)
	assert.Nil(c.BodyBuffer)
	assert.Empty(c.GetHeader(HeaderContentEncoding))
	assert.Empty(c.GetHeader(HeaderContentType))
}

func TestCacheControl(t *testing.T) {
	checkCacheControl := func(resp *httptest.ResponseRecorder, value string, t *testing.T) {
		assert := assert.New(t)
		assert.Equal(value, resp.HeaderMap["Cache-Control"][0])
	}
	t.Run("no cache", func(t *testing.T) {
		resp := httptest.NewRecorder()
		c := NewContext(resp, nil)
		c.NoCache()
		checkCacheControl(resp, "no-cache", t)
	})

	t.Run("no store", func(t *testing.T) {
		resp := httptest.NewRecorder()
		c := NewContext(resp, nil)
		c.NoStore()
		checkCacheControl(resp, "no-store", t)
	})

	t.Run("set cache max age", func(t *testing.T) {
		resp := httptest.NewRecorder()
		c := NewContext(resp, nil)
		c.CacheMaxAge("1m")
		checkCacheControl(resp, "public, max-age=60", t)
	})
	t.Run("set cache s-maxage", func(t *testing.T) {
		resp := httptest.NewRecorder()
		c := NewContext(resp, nil)
		c.CacheSMaxAge("1m", "10s")
		checkCacheControl(resp, "public, max-age=60, s-maxage=10", t)
	})
}

func TestSetContentTypeByExt(t *testing.T) {
	assert := assert.New(t)
	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	headers := c.Header()

	check := func(contentType string) {
		v := headers.Get(HeaderContentType)
		assert.Equal(contentType, v)
	}
	c.SetContentTypeByExt(".html")
	check("text/html; charset=utf-8")
	c.SetHeader(HeaderContentType, "")

	c.SetContentTypeByExt("index.html")
	check("text/html; charset=utf-8")
	c.SetHeader(HeaderContentType, "")

	c.SetContentTypeByExt("")
	check("")
	c.SetHeader(HeaderContentType, "")

	c.SetContentTypeByExt("../abcd/index.html")
	check("text/html; charset=utf-8")
	c.SetHeader(HeaderContentType, "")
}

func TestDisableReuse(t *testing.T) {
	assert := assert.New(t)
	c := &Context{}
	c.DisableReuse()
	assert.True(c.reuseDisabled)
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
	d := New()
	another := New()
	another.GET("/", func(c *Context) error {
		c.BodyBuffer = bytes.NewBufferString("new data")
		return nil
	})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	resp := httptest.NewRecorder()
	d.GET("/", func(c *Context) error {
		c.Pass(another)
		// the data will be ignored
		c.BodyBuffer = bytes.NewBufferString("original data")
		return nil
	})
	d.ServeHTTP(resp, req)
	assert.Equal(http.StatusOK, resp.Code)
	assert.Equal("new data", resp.Body.String())
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
