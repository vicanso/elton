package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/cod"
)

func TestBasicAuth(t *testing.T) {
	m := NewBasicAuth(BasicAuthConfig{
		Validate: func(account, pwd string, c *cod.Context) (bool, error) {
			if account == "tree.xie" || pwd == "password" {
				return true, nil
			}
			return false, nil
		},
	})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)

	t.Run("no auth header", func(t *testing.T) {
		d := cod.New()
		d.Use(m)
		d.GET("/", func(c *cod.Context) error {
			return nil
		})
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("http status code should be 401")
		}
		if resp.Header().Get(cod.HeaderWWWAuthenticate) != "basic realm=basic auth tips" {
			t.Fatalf("www authenticate header is invalid")
		}
	})

	t.Run("auth value not base64", func(t *testing.T) {
		d := cod.New()
		d.Use(m)
		d.GET("/", func(c *cod.Context) error {
			return nil
		})
		req.Header.Set(cod.HeaderAuthorization, "basic 测试")
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest ||
			resp.Body.String() != "category=cod-basic-auth, status=400, message=illegal base64 data at input byte 0" {
			t.Fatalf("base64 decode fail error is invalid")
		}
	})

	t.Run("auth validate fail", func(t *testing.T) {
		d := cod.New()
		d.Use(m)
		d.GET("/", func(c *cod.Context) error {
			return nil
		})
		req.Header.Set(cod.HeaderAuthorization, "basic YTpi")
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized ||
			resp.Body.String() != "category=cod-basic-auth, status=401, message=unAuthorized" {
			t.Fatalf("validate fail error is invalid")
		}
	})

	t.Run("auth success", func(t *testing.T) {
		d := cod.New()
		d.Use(m)
		done := false
		d.GET("/", func(c *cod.Context) error {
			done = true
			return nil
		})
		req.Header.Set(cod.HeaderAuthorization, "basic dHJlZS54aWU6cGFzc3dvcmQ=")
		resp := httptest.NewRecorder()
		d.ServeHTTP(resp, req)
		if !done {
			t.Fatalf("auth fail")
		}
	})
}
