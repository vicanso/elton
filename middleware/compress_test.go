package middleware

import (
	"bytes"
	"errors"
	"math/rand"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vicanso/cod"
)

var letterRunes = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")

type brCompressor struct{}

func (br *brCompressor) Accept(c *cod.Context) (acceptable bool, encoding string) {
	return AcceptEncoding(c, "br")
}

func (br *brCompressor) Compress(buf []byte, level int) ([]byte, error) {
	return []byte("abcd"), nil
}

// randomString get random string
func randomString(n int) string {
	b := make([]rune, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func TestAddGzip(t *testing.T) {
	compressorList := make([]Compressor, 0)
	compressorList = addGzip(compressorList)
	compressorList = addGzip(compressorList)
	if len(compressorList) != 2 {
		t.Fatalf("add gzip fail")
	}
}

func TestCompress(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		c := cod.NewContext(nil, nil)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		fn := NewCompress(CompressConfig{
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

	t.Run("nil body", func(t *testing.T) {
		c := cod.NewContext(nil, nil)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		fn := NewDefaultCompress()
		err := fn(c)
		if err != nil ||
			!done {
			t.Fatalf("nil body should skip")
		}
	})

	t.Run("return error", func(t *testing.T) {
		c := cod.NewContext(nil, nil)
		customErr := errors.New("abccd")
		c.Next = func() error {
			return customErr
		}
		fn := NewCompress(CompressConfig{})
		err := fn(c)
		if err != customErr {
			t.Fatalf("it should return error")
		}
	})

	t.Run("normal", func(t *testing.T) {
		fn := NewCompress(CompressConfig{
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
		fn := NewCompress(CompressConfig{})
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
		fn := NewCompress(CompressConfig{})

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
		fn := NewCompress(CompressConfig{})

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
		fn := NewCompress(CompressConfig{})

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

		compressorList := make([]Compressor, 0)
		compressorList = append(compressorList, new(brCompressor))
		fn := NewCompress(CompressConfig{
			CompressorList: compressorList,
		})

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
