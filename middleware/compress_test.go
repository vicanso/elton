package middleware

import (
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
}
