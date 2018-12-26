package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/cod"
)

func TestFresh(t *testing.T) {
	fn := NewFresh(FreshConfig{})
	modifiedAt := "Tue, 25 Dec 2018 00:02:22 GMT"
	t.Run("modified", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(cod.HeaderIfModifiedSince, modifiedAt)
		resp := httptest.NewRecorder()
		resp.Header().Set(cod.HeaderLastModified, modifiedAt)

		c := cod.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			c.StatusCode = http.StatusOK
			c.Body = map[string]string{
				"name": "tree.xie",
			}
			c.BodyBytes = []byte(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		if err != nil || !done {
			t.Fatalf("fresh middleware fail, %v", err)
		}

		if c.StatusCode != 304 ||
			c.Body != nil ||
			c.BodyBytes != nil {
			t.Fatalf("fresh middleware response fail")
		}
	})

	t.Run("no body", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(cod.HeaderIfModifiedSince, modifiedAt)
		resp := httptest.NewRecorder()
		resp.Header().Set(cod.HeaderLastModified, modifiedAt)
		c := cod.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		c.NoContent()
		err := fn(c)
		if err != nil {
			t.Fatalf("fresh middleware fail, %v", err)
		}
		if c.StatusCode == 304 {
			t.Fatalf("no body should pass fresh middleware")
		}
	})

	t.Run("post method", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users/me", nil)
		req.Header.Set(cod.HeaderIfModifiedSince, modifiedAt)
		resp := httptest.NewRecorder()
		resp.Header().Set(cod.HeaderLastModified, modifiedAt)

		c := cod.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			c.StatusCode = http.StatusOK
			c.Body = map[string]string{
				"name": "tree.xie",
			}
			c.BodyBytes = []byte(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		if err != nil || !done {
			t.Fatalf("fresh middleware fail, %v", err)
		}

		if c.StatusCode == 304 ||
			c.Body == nil ||
			c.BodyBytes == nil {
			t.Fatalf("fresh middleware response fail")
		}
	})

	t.Run("error response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(cod.HeaderIfModifiedSince, modifiedAt)
		resp := httptest.NewRecorder()
		resp.Header().Set(cod.HeaderLastModified, modifiedAt)

		c := cod.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			c.StatusCode = http.StatusBadRequest
			c.Body = map[string]string{
				"name": "tree.xie",
			}
			c.BodyBytes = []byte(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		if err != nil || !done {
			t.Fatalf("fresh middleware fail, %v", err)
		}

		if c.StatusCode == 304 ||
			c.Body == nil ||
			c.BodyBytes == nil {
			t.Fatalf("fresh middleware response fail")
		}
	})

}
