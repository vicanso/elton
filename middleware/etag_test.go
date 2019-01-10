package middleware

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/cod"
)

func TestETag(t *testing.T) {
	fn := NewETag(ETagConfig{})
	t.Run("no body", func(t *testing.T) {
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, nil)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		if err != nil {
			t.Fatalf("eTag middleware fail, %v", err)
		}
		if c.GetHeader(cod.HeaderETag) != "" {
			t.Fatalf("no body should not gen eTag")
		}
	})

	t.Run("error status", func(t *testing.T) {
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, nil)
		c.Next = func() error {
			c.Body = map[string]string{
				"name": "tree.xie",
			}
			c.StatusCode = 400
			c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		if err != nil {
			t.Fatalf("eTag middleware fail, %v", err)
		}
		if c.GetHeader(cod.HeaderETag) != "" {
			t.Fatalf("error status should not gen eTag")
		}
	})

	t.Run("gen eTag", func(t *testing.T) {
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, nil)
		c.Next = func() error {
			c.Body = map[string]string{
				"name": "tree.xie",
			}
			c.StatusCode = 200
			c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		if err != nil {
			t.Fatalf("eTag middleware fail, %v", err)
		}
		if c.GetHeader(cod.HeaderETag) != `"13-yo9YroUOjW1obRvVoXfrCiL2JGE="` {
			t.Fatalf("gen eTag fail")
		}
	})
}
