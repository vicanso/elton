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
		c.SetHeader(cod.HeaderContentType, "text/html")
		c.BodyBuffer = bytes.NewBuffer([]byte("<html><body>" + randomString(8192) + "</body></html>"))
		originalSize := c.BodyBuffer.Len()
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		if err != nil || !done {
			t.Fatalf("compress middleware fail, %v", err)
		}
		if c.BodyBuffer.Len() >= originalSize ||
			c.GetHeader(cod.HeaderContentEncoding) != "gzip" {
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
		body := bytes.NewBufferString(randomString(4096))
		c.BodyBuffer = body
		c.SetHeader(cod.HeaderContentEncoding, "gzip")
		err := fn(c)
		if err != nil {
			t.Fatalf("compress fail, %v", err)
		}
		if !bytes.Equal(c.BodyBuffer.Bytes(), body.Bytes()) {
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
		body := bytes.NewBufferString("abcd")
		c.BodyBuffer = body
		err := fn(c)
		if err != nil {
			t.Fatalf("compress fail, %v", err)
		}
		if !bytes.Equal(c.BodyBuffer.Bytes(), body.Bytes()) ||
			c.GetHeader(cod.HeaderContentEncoding) != "" {
			t.Fatalf("less than min length should not be compress")
		}
	})

	t.Run("image should not be compress", func(t *testing.T) {
		fn := NewCompresss(CompressConfig{})

		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(cod.HeaderAcceptEncoding, "gzip")
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		c.SetHeader(cod.HeaderContentType, "image/jpeg")
		c.Next = func() error {
			return nil
		}
		body := bytes.NewBufferString(randomString(4096))
		c.BodyBuffer = body
		err := fn(c)
		if err != nil {
			t.Fatalf("compress fail, %v", err)
		}
		if !bytes.Equal(c.BodyBuffer.Bytes(), body.Bytes()) ||
			c.GetHeader(cod.HeaderContentEncoding) != "" {
			t.Fatalf("image should not be compress")
		}
	})

	t.Run("not accept gzip should not compress", func(t *testing.T) {
		fn := NewCompresss(CompressConfig{})

		req := httptest.NewRequest("GET", "/users/me", nil)
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		c.SetHeader(cod.HeaderContentType, "text/html")
		c.Next = func() error {
			return nil
		}
		body := bytes.NewBufferString(randomString(4096))
		c.BodyBuffer = body
		err := fn(c)
		if err != nil {
			t.Fatalf("compress fail, %v", err)
		}
		if !bytes.Equal(c.BodyBuffer.Bytes(), body.Bytes()) ||
			c.GetHeader(cod.HeaderContentEncoding) != "" {
			t.Fatalf("not accept gzip should not be compress")
		}
	})

	t.Run("custom compress", func(t *testing.T) {
		brCompress := &Compression{
			Type: "br",
			Compress: func(buf []byte, level int) ([]byte, error) {
				return []byte("abcd"), nil
			},
		}
		compressionList := make([]*Compression, 0)
		compressionList = append(compressionList, brCompress)
		fn := NewCompresss(CompressConfig{
			CompressionList: compressionList,
		})
		// fn := NewCompresss(CompressConfig{
		// 	Compresss: func(c *cod.Context) (done bool) {
		// 		// 假设做了 brotli 压缩
		// 		c.BodyBuffer = bytes.NewBufferString("abcd")
		// 		c.SetHeader(cod.HeaderContentEncoding, "br")
		// 		return true
		// 	},
		// })

		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		c.SetHeader(cod.HeaderContentType, "text/html")
		c.BodyBuffer = bytes.NewBufferString("<html><body>" + randomString(8192) + "</body></html>")
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		if err != nil || !done {
			t.Fatalf("compress middleware fail, %v", err)
		}
		if c.BodyBuffer.Len() != 4 ||
			c.GetHeader(cod.HeaderContentEncoding) != "br" {
			t.Fatalf("custom compress fail")
		}
	})
}
