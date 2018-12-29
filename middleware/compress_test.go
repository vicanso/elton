package middleware

import (
	"bytes"
	"math/rand"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vicanso/cod"
)

var letterRunes = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")

// randomString get random string
func randomString(n int) string {
	b := make([]rune, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func TestCompress(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		fn := NewCompresss(CompressConfig{
			Level:     1,
			MinLength: 1,
		})

		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(cod.HeaderAcceptEncoding, "gzip")
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		c.Headers.Set(cod.HeaderContentType, "text/html")
		c.BodyBytes = []byte("<html><body>" + randomString(8192) + "</body></html>")
		originalSize := len(c.BodyBytes)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		if err != nil || !done {
			t.Fatalf("compress middleware fail, %v", err)
		}
		if len(c.BodyBytes) >= originalSize {
			t.Fatalf("compress fail")
		}
	})

	t.Run("encoding done", func(t *testing.T) {
		fn := NewCompresss(CompressConfig{})
		req := httptest.NewRequest("GET", "/users/me", nil)
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		body := []byte(randomString(4096))
		c.BodyBytes = body
		c.Headers.Set(cod.HeaderContentEncoding, "gzip")
		err := fn(c)
		if err != nil {
			t.Fatalf("compress fail, %v", err)
		}
		if !bytes.Equal(c.BodyBytes, body) {
			t.Fatalf("the data is encoding, it should not be compress")
		}
	})

	t.Run("body size is less than min length", func(t *testing.T) {
		fn := NewCompresss(CompressConfig{})

		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(cod.HeaderAcceptEncoding, "gzip")
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		body := []byte("abcd")
		c.BodyBytes = body
		err := fn(c)
		if err != nil {
			t.Fatalf("compress fail, %v", err)
		}
		if !bytes.Equal(c.BodyBytes, body) ||
			c.Headers.Get(cod.HeaderContentEncoding) != "" {
			t.Fatalf("less than min length should not be compress")
		}
	})

	t.Run("image should not be compress", func(t *testing.T) {
		fn := NewCompresss(CompressConfig{})

		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(cod.HeaderAcceptEncoding, "gzip")
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		c.Headers.Set(cod.HeaderContentType, "image/jpeg")
		c.Next = func() error {
			return nil
		}
		body := []byte(randomString(4096))
		c.BodyBytes = body
		err := fn(c)
		if err != nil {
			t.Fatalf("compress fail, %v", err)
		}
		if !bytes.Equal(c.BodyBytes, body) ||
			c.Headers.Get(cod.HeaderContentEncoding) != "" {
			t.Fatalf("image should not be compress")
		}
	})

	t.Run("not accept gzip should not compress", func(t *testing.T) {
		fn := NewCompresss(CompressConfig{})

		req := httptest.NewRequest("GET", "/users/me", nil)
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		c.Headers.Set(cod.HeaderContentType, "text/html")
		c.Next = func() error {
			return nil
		}
		body := []byte(randomString(4096))
		c.BodyBytes = body
		err := fn(c)
		if err != nil {
			t.Fatalf("compress fail, %v", err)
		}
		if !bytes.Equal(c.BodyBytes, body) ||
			c.Headers.Get(cod.HeaderContentEncoding) != "" {
			t.Fatalf("not accept gzip should not be compress")
		}
	})
}
