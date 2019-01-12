package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/vicanso/cod"
)

func TestProxy(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		target, _ := url.Parse("https://www.baidu.com")
		config := ProxyConfig{
			Target:    target,
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

	t.Run("target picker", func(t *testing.T) {
		target, _ := url.Parse("https://www.baidu.com")
		config := ProxyConfig{
			TargetPicker: func(c *cod.Context) (*url.URL, error) {
				return target, nil
			},
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
