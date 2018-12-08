package middleware

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/cod"
)

func checkResponse(t *testing.T, resp *httptest.ResponseRecorder, code int, data string) {
	if resp.Body.String() != data ||
		resp.Code != code {
		t.Fatalf("check response fail")
	}
}

func checkJSON(t *testing.T, resp *httptest.ResponseRecorder) {
	if resp.Header().Get(cod.HeaderContentType) != cod.MIMEApplicationJSON {
		t.Fatalf("response content type should be json")
	}
}

func checkContentType(t *testing.T, resp *httptest.ResponseRecorder, contentType string) {
	if resp.Header().Get(cod.HeaderContentType) != contentType {
		t.Fatalf("response content type check fail")
	}
}

func TestResponder(t *testing.T) {
	m := NewResponder(ResponderConfig{})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)

	t.Run("invalid response", func(t *testing.T) {
		d := cod.New()
		d.Use(m)
		d.GET("/", func(c *cod.Context) error {
			return nil
		})
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		checkResponse(t, resp, 500, `{"status_code":500,"category":"cod","message":"invalid response"}`)
		checkJSON(t, resp)
	})

	t.Run("return error", func(t *testing.T) {
		d := cod.New()
		d.Use(m)
		d.GET("/", func(c *cod.Context) error {
			return errors.New("abc")
		})
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		checkResponse(t, resp, 500, `{"status_code":500,"message":"abc"}`)
		checkJSON(t, resp)
	})

	t.Run("return http error", func(t *testing.T) {
		d := cod.New()
		d.Use(m)
		d.GET("/", func(c *cod.Context) error {
			return &cod.HTTPError{
				StatusCode: 400,
				Message:    "abc",
				Category:   "custom",
			}
		})
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		checkResponse(t, resp, 400, `{"status_code":400,"category":"custom","message":"abc"}`)
		checkJSON(t, resp)
	})

	t.Run("return string", func(t *testing.T) {
		d := cod.New()
		d.Use(m)
		d.GET("/", func(c *cod.Context) error {
			c.Body = "abc"
			return nil
		})
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		checkResponse(t, resp, 200, "abc")
		checkContentType(t, resp, cod.MIMETextPlain)
	})

	t.Run("return bytes", func(t *testing.T) {
		d := cod.New()
		d.Use(m)
		d.GET("/", func(c *cod.Context) error {
			c.Body = []byte("abc")
			return nil
		})
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		checkResponse(t, resp, 200, "abc")
		checkContentType(t, resp, cod.MIMEBinary)
	})

	t.Run("return struct", func(t *testing.T) {
		type T struct {
			Name string `json:"name,omitempty"`
		}
		d := cod.New()
		d.Use(m)
		d.GET("/", func(c *cod.Context) error {
			c.Created(&T{
				Name: "tree.xie",
			})
			return nil
		})
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		checkResponse(t, resp, 201, `{"name":"tree.xie"}`)
		checkJSON(t, resp)
	})

	t.Run("json marshal fail", func(t *testing.T) {
		d := cod.New()
		d.Use(m)
		d.GET("/", func(c *cod.Context) error {
			c.Body = func() {}
			return nil
		})
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		checkResponse(t, resp, 500, "func() is unsupported type")
	})
}
