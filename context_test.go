package cod

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReset(t *testing.T) {
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
		Params:      make(map[string]string),
		StatusCode:  200,
		Body:        make(map[string]string),
		BodyBuffer:  bytes.NewBufferString("abcd"),
		RequestBody: []byte("abcd"),
		m:           make(map[string]interface{}),
		realIP:      "abcd",
		cod:         &Cod{},
	}
	c.Reset()
	if c.Request != nil ||
		c.Response != nil ||
		c.Headers != nil ||
		c.Committed ||
		c.ID != "" ||
		c.Route != "" ||
		c.Next != nil ||
		c.Params != nil ||
		c.StatusCode != 0 ||
		c.Body != nil ||
		c.BodyBuffer != nil ||
		c.RequestBody != nil ||
		c.m != nil ||
		c.realIP != "" ||
		c.cod != nil {
		t.Fatalf("reset fail")
	}
}

func TestRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)

	c := Context{
		Request: req,
	}
	t.Run("get from x-forwarded-for", func(t *testing.T) {
		defer req.Header.Del(HeaderXForwardedFor)
		req.Header.Set(HeaderXForwardedFor, "192.0.0.1, 192.168.1.1")
		if c.RealIP() != "192.0.0.1" {
			t.Fatalf("get real ip from x-forwarded-for fail")
		}
		c.realIP = ""
	})

	t.Run("get from x-real-ip", func(t *testing.T) {
		defer req.Header.Del(HeaderXRealIp)
		req.Header.Set(HeaderXRealIp, "192.168.0.1")
		if c.RealIP() != "192.168.0.1" {
			t.Fatalf("get real ip from x-real-ip fail")
		}
		c.realIP = ""
	})

	t.Run("get real ip from remote addr", func(t *testing.T) {
		if c.RealIP() == "" {
			t.Fatalf("get real ip from remote addr fail")
		}
		c.realIP = ""
	})
}

func TestParam(t *testing.T) {
	c := Context{}
	if c.Param("name") != "" {
		t.Fatalf("params is not initialized, it should be nil")
	}
	c.Params = map[string]string{
		"name": "tree.xie",
	}
	if c.Param("name") != "tree.xie" {
		t.Fatalf("get param fail")
	}
}

func TestQueryParam(t *testing.T) {
	req := httptest.NewRequest("GET", "https://aslant.site/?name=tree.xie", nil)
	resp := httptest.NewRecorder()
	c := NewContext(resp, req)
	if c.QueryParam("name") != "tree.xie" {
		t.Fatalf("get query fail")
	}

	if c.QueryParam("account") != "" {
		t.Fatalf("get not exists query fail")
	}
}

func TestQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "https://aslant.site/?name=tree.xie&type=1", nil)
	resp := httptest.NewRecorder()
	c := NewContext(resp, req)
	q := c.Query()
	if q["name"] != "tree.xie" ||
		q["type"] != "1" {
		t.Fatalf("get query fail")
	}
}

func TestSetGet(t *testing.T) {
	c := Context{}
	if c.Get("name") != nil {
		t.Fatalf("should return nil when store is not initialized")
	}
	c.Set("name", "tree.xie")
	if c.Get("name").(string) != "tree.xie" {
		t.Fatalf("set/get fail")
	}
}

func TestGetSetHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "https://aslant.site/?name=tree.xie&type=1", nil)
	req.Header.Set("X-Token", "abc")
	resp := httptest.NewRecorder()
	c := NewContext(resp, req)

	t.Run("get header from request", func(t *testing.T) {
		if c.GetRequestHeader("X-Token") != "abc" {
			t.Fatalf("get header from request fail")
		}
	})

	t.Run("set header to the response", func(t *testing.T) {
		c.SetHeader("X-Response-Id", "1")
		if c.GetHeader("X-Response-Id") != "1" {
			t.Fatalf("set header to response fail")
		}
	})

	t.Run("get header from response", func(t *testing.T) {
		idc := "GZ"
		key := "X-IDC"
		c.SetHeader(key, idc)
		if c.GetHeader(key) != idc {
			t.Fatalf("get header from response fail")
		}
	})

	t.Run("get header of response", func(t *testing.T) {
		if c.Header() == nil {
			t.Fatalf("header function fail")
		}
	})
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
		cookie, err := c.Cookie("a")
		if err != nil {
			t.Fatalf("get cookie fail, %v", err)
		}
		if cookie.Name != "a" ||
			cookie.Value != "b" {
			t.Fatalf("get cookie fail")
		}
	})

	t.Run("set cookie", func(t *testing.T) {
		cookie := &http.Cookie{
			Name:     "a",
			Value:    "b",
			MaxAge:   300,
			Secure:   true,
			Path:     "/",
			HttpOnly: true,
		}
		c.SetCookie(cookie)
		if c.GetHeader(HeaderSetCookie) != "a=b; Path=/; Max-Age=300; HttpOnly; Secure" {
			t.Fatalf("set cookie fail")
		}
	})
}

func TestRedirect(t *testing.T) {
	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	err := c.Redirect(299, "")
	if err != ErrInvalidRedirect {
		t.Fatalf("invalid redirect code should return error")
	}

	url := "https://aslant.site/"
	err = c.Redirect(302, url)
	if err != nil {
		t.Fatalf("redirect fail, %v", err)
	}
	if c.GetHeader(HeaderLocation) != url {
		t.Fatalf("set location fail")
	}
}

func TestCreate(t *testing.T) {
	body := "abc"
	c := NewContext(nil, nil)
	c.Created(body)
	if c.StatusCode != http.StatusCreated ||
		c.Body.(string) != body {
		t.Fatalf("create for response fail")
	}
}

func TestNoContent(t *testing.T) {
	c := NewContext(nil, nil)
	c.NoContent()
	if c.StatusCode != http.StatusNoContent {
		t.Fatalf("set no content fail")
	}
}

func TestNotModified(t *testing.T) {
	c := NewContext(nil, nil)
	c.NotModified()
	if c.StatusCode != http.StatusNotModified {
		t.Fatalf("set not modified fail")
	}
}

func TestCacheControl(t *testing.T) {
	checkCacheControl := func(resp *httptest.ResponseRecorder, value string, t *testing.T) {
		if resp.HeaderMap["Cache-Control"][0] != value {
			t.Fatalf("cache control should be " + value)
		}
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
}

func TestSetFileContentType(t *testing.T) {
	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	headers := c.Header()

	check := func(contentType string) {
		v := headers.Get(HeaderContentType)
		if v != contentType {
			t.Fatalf("content type should be %s, but %s", contentType, v)
		}
	}
	c.SetFileContentType(".html")
	check("text/html; charset=utf-8")
	c.SetHeader(HeaderContentType, "")

	c.SetFileContentType("index.html")
	check("text/html; charset=utf-8")
	c.SetHeader(HeaderContentType, "")

	c.SetFileContentType("")
	check("")
	c.SetHeader(HeaderContentType, "")

	c.SetFileContentType("../abcd/index.html")
	check("text/html; charset=utf-8")
	c.SetHeader(HeaderContentType, "")

}

func TestPush(t *testing.T) {
	resp := httptest.NewRecorder()
	c := NewContext(resp, nil)
	err := c.Push("/a.css", nil)
	if err != nil {
		t.Fatalf("push fail, %v", err)
	}
}

func TestGetCod(t *testing.T) {
	c := NewContext(nil, nil)
	c.cod = &Cod{}
	if c.Cod() == nil {
		t.Fatalf("get cod instance fail")
	}
}
func TestNewContext(t *testing.T) {
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	resp := httptest.NewRecorder()
	c := NewContext(resp, req)
	if c.Request != req ||
		c.Response != resp {
		t.Fatalf("new context fail")
	}
}
