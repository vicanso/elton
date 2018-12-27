package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/cod"
)

func TestProxy(t *testing.T) {
	config := ProxyConfig{
		URL:  "https://www.baidu.com",
		Host: "www.baidu.com",
	}
	fn := NewProxy(config)
	req := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
	resp := httptest.NewRecorder()
	c := cod.NewContext(resp, req)
	fn(c)
	if resp.Code != http.StatusOK {
		t.Fatalf("http proxy fail")
	}
}
