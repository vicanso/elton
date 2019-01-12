package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/cod"
)

func TestRecover(t *testing.T) {
	var ctx *cod.Context
	d := cod.New()
	d.Use(NewRecover())
	d.GET("/", func(c *cod.Context) error {
		ctx = c
		panic("abc")
	})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)
	resp := httptest.NewRecorder()
	d.ServeHTTP(resp, req)
	if resp.Code != http.StatusInternalServerError ||
		resp.Body.String() != "abc" ||
		!ctx.Committed {
		t.Fatalf("recover fail")
	}
}
