package cod

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkRoutes(b *testing.B) {
	d := NewWithoutServer()
	d.GET("/", func(c *Context) error {
		return nil
	})
	b.ReportAllocs()
	req := httptest.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		d.ServeHTTP(resp, req)
	}
}

func BenchmarkGetFunctionName(b *testing.B) {
	b.ReportAllocs()
	fn := func() {}
	d := New()
	d.SetFunctionName(fn, "test-fn")
	for i := 0; i < b.N; i++ {
		d.GetFunctionName(fn)
	}
}
