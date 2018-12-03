package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/cod"
)

func TestRecover(t *testing.T) {
	d := cod.New()
	d.Use(NewRecover())
	d.GET("/", func(c *cod.Context) error {
		panic("abc")
	})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	if resp.Code != http.StatusInternalServerError ||
		resp.Body.String() != "abc" {
		t.Fatalf("recover fail")
	}
}
