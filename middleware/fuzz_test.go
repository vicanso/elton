// MIT License

// Copyright (c) 2026 Tree Xie

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

package middleware

import (
	"bytes"
	"net/http"
	"testing"
)

// FuzzNewCacheResponse 确保解码任意（含损坏/截断）的缓存数据不会panic，
// 自定义store（如Redis）可能返回异常内容
func FuzzNewCacheResponse(f *testing.F) {
	f.Add([]byte(nil))
	f.Add(hitForPassData)
	// 一份合法的hit缓存数据作为种子
	valid := (&CacheResponse{
		Status:     StatusHit,
		CreatedAt:  1700000000,
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
		},
		Body: bytes.NewBufferString(`{"name":"elton"}`),
	}).Bytes()
	f.Add(valid)
	f.Fuzz(func(t *testing.T, data []byte) {
		resp := NewCacheResponse(data)
		if resp == nil {
			t.Fatal("cache response should not be nil")
		}
	})
}

// FuzzHTTPHeadersHeader 确保解码任意字节序列的header数据不会panic
func FuzzHTTPHeadersHeader(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte(NewHTTPHeaders(http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Custom":     []string{"a", "b"},
	})))
	f.Fuzz(func(t *testing.T, data []byte) {
		_ = HTTPHeaders(data).Header()
	})
}
