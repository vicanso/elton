package middleware

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/cod"
)

func TestFresh(t *testing.T) {
	fn := NewDefaultFresh()
	modifiedAt := "Tue, 25 Dec 2018 00:02:22 GMT"
	t.Run("skip", func(t *testing.T) {
		c := cod.NewContext(nil, nil)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		fn := NewFresh(FreshConfig{
			Skipper: func(c *cod.Context) bool {
				return true
			},
		})
		err := fn(c)
		if err != nil ||
			!done {
			t.Fatalf("skip fail")
		}
	})

	t.Run("return error", func(t *testing.T) {
		c := cod.NewContext(nil, nil)
		customErr := errors.New("abccd")
		c.Next = func() error {
			return customErr
		}
		fn := NewFresh(FreshConfig{})
		err := fn(c)
		if err != customErr {
			t.Fatalf("it should return error")
		}
	})

	t.Run("not modified", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(cod.HeaderIfModifiedSince, modifiedAt)
		resp := httptest.NewRecorder()
		resp.Header().Set(cod.HeaderLastModified, modifiedAt)

		c := cod.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			c.Body = map[string]string{
				"name": "tree.xie",
			}
			c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		if err != nil || !done {
			t.Fatalf("fresh middleware fail, %v", err)
		}

		if c.StatusCode != 304 ||
			c.Body != nil ||
			c.BodyBuffer != nil {
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
			c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		if err != nil || !done {
			t.Fatalf("fresh middleware fail, %v", err)
		}

		if c.StatusCode == 304 ||
			c.Body == nil ||
			c.BodyBuffer == nil {
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
			c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		if err != nil || !done {
			t.Fatalf("fresh middleware fail, %v", err)
		}

		if c.StatusCode == 304 ||
			c.Body == nil ||
			c.BodyBuffer == nil {
			t.Fatalf("fresh middleware response fail")
		}
	})

}
