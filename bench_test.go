package cod

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

func BenchmarkGetStatus(b *testing.B) {
	b.ReportAllocs()
	var v int32
	for i := 0; i < b.N; i++ {
		atomic.LoadInt32(&v)
	}
}

func BenchmarkBufferPick(b *testing.B) {
	b.ReportAllocs()
	b64 := base64.StdEncoding.EncodeToString(make([]byte, 1024))
	m := map[string]interface{}{
		"_x": b64,
		"_y": b64,
		"_z": b64,
		"i":  1,
		"f":  1.12,
		"s":  "\"abc",
		"b":  false,
		"arr": []interface{}{
			1,
			"2",
			true,
		},
		"m": map[string]interface{}{
			"a": 1,
			"b": "2",
			"c": false,
		},
		"null": nil,
	}
	buf, _ := json.Marshal(m)
	for i := 0; i < b.N; i++ {
		JSONPick(buf, strings.Split("i,f,s,b,arr,m,null", ","))
	}
}

func BenchmarkCamelCase(b *testing.B) {
	b.ReportAllocs()
	str := "Foo Bar"
	for i := 0; i < b.N; i++ {
		CamelCase(str)
	}
}

func BenchmarkCamelCaseJSON(b *testing.B) {
	b.ReportAllocs()
	json := []byte(`{
		"book_name": "test",
		"book_price": 12,
		"book_on_sale": true,
		"book_author": {
			"author_name": "tree.xie",
			"author_age": 0,
			"author_salary": 10.1,
		},
		"book_category": ["vip", "hot-sale"],
		"book_infos": [
			{
				"word_count": 100
			}
		]
	}`)
	for i := 0; i < b.N; i++ {
		CamelCaseJSON(json)
	}
}
