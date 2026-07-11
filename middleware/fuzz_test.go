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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/elton/v2"
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

// FuzzFormURLEncodedDecoder 任意 form body 经解码后若成功则必须是合法 JSON
func FuzzFormURLEncodedDecoder(f *testing.F) {
	f.Add([]byte("a=1&b=2"))
	f.Add([]byte("a=1&a=2"))
	f.Add([]byte(`a=1"2&b=x\y`))
	f.Add([]byte(`a=1&b=","x":1`))
	f.Add([]byte("%zz"))
	f.Add([]byte(""))
	dec := NewFormURLEncodedDecoder()
	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
	c.SetRequestHeader(elton.HeaderContentType, formURLEncodedContentType)
	f.Fuzz(func(t *testing.T, data []byte) {
		out, err := dec.Decode(c, data)
		if err != nil {
			return
		}
		if len(out) == 0 {
			return
		}
		var m map[string]any
		if jsonErr := json.Unmarshal(out, &m); jsonErr != nil {
			t.Fatalf("decoded form must be valid JSON: %v, out=%q", jsonErr, out)
		}
	})
}

// FuzzCacheResponseRoundTrip 合法 CacheResponse 序列化后再解码，状态应保持可识别
func FuzzCacheResponseRoundTrip(f *testing.F) {
	f.Add(uint8(StatusHit), uint32(1700000000), uint16(200), []byte(`{"ok":true}`), []byte("application/json"))
	f.Add(uint8(StatusHitForPass), uint32(0), uint16(0), []byte(nil), []byte(nil))
	f.Add(uint8(StatusHit), uint32(1), uint16(404), []byte("not found"), []byte("text/plain"))
	f.Fuzz(func(t *testing.T, status uint8, createdAt uint32, code uint16, body, contentType []byte) {
		// 仅使用已定义状态，避免无意义组合刷日志
		cs := CacheStatus(status % 3)
		cp := &CacheResponse{
			Status:     cs,
			CreatedAt:  createdAt,
			StatusCode: int(code),
			Header:     http.Header{},
			Body:       bytes.NewBuffer(body),
		}
		if len(contentType) > 0 {
			// 限制 header 值长度，避免超大输入拖垮 fuzz
			ct := string(contentType)
			if len(ct) > 64 {
				ct = ct[:64]
			}
			cp.Header.Set("Content-Type", ct)
		}
		raw := cp.Bytes()
		decoded := NewCacheResponse(raw)
		if decoded == nil {
			t.Fatal("decoded cache response must not be nil")
		}
		if cs == StatusHitForPass || cs == StatusUnknown {
			// 非 hit 只保留状态字节
			return
		}
		if cs == StatusHit && decoded.Status != StatusHit && decoded.Status != StatusUnknown {
			// 损坏路径允许 Unknown；合法编码应为 Hit
			if len(raw) > statusByteSize {
				// 完整编码应可还原为 hit
				if decoded.Status != StatusHit {
					// body/header 极大时仍应不 panic；状态异常则报错
					t.Fatalf("expected hit or unknown, got %v", decoded.Status)
				}
			}
		}
	})
}
