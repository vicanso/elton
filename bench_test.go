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
		_ = make(map[string]interface{})
	}
}

func BenchmarkConvertServerTiming(b *testing.B) {
	b.ReportAllocs()
	traceInfos := make(TraceInfos, 0)
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

func BenchmarkGetStatus(b *testing.B) {
	b.ReportAllocs()
	var v int32
	for i := 0; i < b.N; i++ {
		atomic.LoadInt32(&v)
	}
}
