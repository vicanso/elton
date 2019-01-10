package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/cod"
)

func TestProxy(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		config := ProxyConfig{
			URL:       "https://www.baidu.com",
			Host:      "www.baidu.com",
			Transport: &http.Transport{},
		}
		fn := NewProxy(config)
		req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		fn(c)
		if !done || c.StatusCode != http.StatusOK {
			t.Fatalf("http proxy fail")
		}
	})
}
