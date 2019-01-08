package cod

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func BenchmarkContextGet(b *testing.B) {
	b.ReportAllocs()
	key := "id"
	c := NewContext(nil, nil)

	for i := 0; i < b.N; i++ {
		c.Set(key, "abc")
		_ = c.Get(key).(string)
	}
}

func BenchmarkContextNewMap(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = make(map[string]interface{})
	}
}

func BenchmarkConvertServerTiming(b *testing.B) {
	b.ReportAllocs()
	traceInfos := make([]*TraceInfo, 0)
	for _, name := range strings.Split("0123456789", "") {
		traceInfos = append(traceInfos, &TraceInfo{
			Name:     name,
			Duration: time.Microsecond * 100,
		})
	}
	for i := 0; i < b.N; i++ {
		ConvertToServerTiming(traceInfos, "cod-")
	}
}
