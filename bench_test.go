// MIT License

// Copyright (c) 2020 Tree Xie

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package elton

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkRoutes(b *testing.B) {
	e := NewWithoutServer()
	e.GET("/", func(c *Context) error {
		return nil
	})
	b.ReportAllocs()
	req := httptest.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		e.ServeHTTP(resp, req)
	}
}

func BenchmarkGetFunctionName(b *testing.B) {
	b.ReportAllocs()
	fn := func() {}
	e := New()
	e.SetFunctionName(fn, "test-fn")
	for i := 0; i < b.N; i++ {
		e.GetFunctionName(fn)
	}
}

func BenchmarkContextGet(b *testing.B) {
	b.ReportAllocs()
	key := "id"
	c := NewContext(nil, nil)

	for i := 0; i < b.N; i++ {
		c.Set(key, "abc")
		_, _ = c.Get(key)
	}
}

func BenchmarkContextNewMap(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = make(map[string]any)
	}
}

func BenchmarkConvertServerTiming(b *testing.B) {
	b.ReportAllocs()
	traceInfos := make(TraceInfos, 0, 10)
	for _, name := range strings.Split("0123456789", "") {
		traceInfos = append(traceInfos, &TraceInfo{
			Name:     name,
			Duration: time.Microsecond * 100,
		})
	}
	for i := 0; i < b.N; i++ {
		traceInfos.ServerTiming("elton-")
	}
}

// BenchmarkSignedCookie 测量签名 cookie 校验路径（含 keygrip 缓存）
func BenchmarkSignedCookie(b *testing.B) {
	sk := new(AtomicSignedKeys)
	sk.SetKeys([]string{"secret"})
	e := &Elton{SignedKeys: sk}
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "a", Value: "b"})
	req.AddCookie(&http.Cookie{Name: "a.sig", Value: "jK8pWDfgnIdsDF73KVgdXnXvk63BBCDOcaqwVjasY-0"})
	c := NewContext(httptest.NewRecorder(), req)
	c.elton = e
	// 预热缓存
	_, _ = c.SignedCookie("a")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.SignedCookie("a")
	}
}

// BenchmarkTraceHandlers 测量 EnableTrace 下多中间件名称解析开销
func BenchmarkTraceHandlers(b *testing.B) {
	e := NewWithoutServer()
	e.EnableTrace = true
	e.Use(func(c *Context) error { return c.Next() })
	e.Use(func(c *Context) error { return c.Next() })
	e.Use(func(c *Context) error { return c.Next() })
	e.GET("/", func(c *Context) error {
		c.BodyBuffer = nil
		return nil
	})
	req := httptest.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.ServeHTTP(resp, req)
	}
}

func BenchmarkStatus(b *testing.B) {
	b.ReportAllocs()
	var v int32
	for i := 0; i < b.N; i++ {
		atomic.LoadInt32(&v)
	}
}
