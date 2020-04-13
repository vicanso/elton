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

package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestGetHumanReadableSize(t *testing.T) {
	if getHumanReadableSize(1024*1024) != "1MB" {
		t.Fatalf("1024 * 1024 should be 1MB")
	}
	if getHumanReadableSize(1024*1024+500*1024) != "1.49MB" {
		t.Fatalf("1024*1024+500*1024 should be 1.49MB")
	}

	if getHumanReadableSize(1024) != "1KB" {
		t.Fatalf("1024 should be 1KB")
	}
	if getHumanReadableSize(1024+500) != "1.49KB" {
		t.Fatalf("1024+500 should be 1.49KB")
	}
	if getHumanReadableSize(500) != "500B" {
		t.Fatalf("500 should be 500B")
	}
}

func TestLogger(t *testing.T) {
	assert := assert.New(t)
	t.Run("normal", func(t *testing.T) {
		config := LoggerConfig{
			Format: "{host} {remote} {real-ip} {method} {path} {proto} {query} {scheme} {uri} {referer} {userAgent} {size} {size-human} {status} {payload-size} {payload-size-human}",
			OnLog: func(log string, _ *elton.Context) {
				if log != "aslant.site 192.0.2.1:1234 192.0.2.1 GET / HTTP/1.1 a=1&b=2 HTTPS https://aslant.site/?a=1&b=2 https://aslant.site/ test-agent 13 13B 200 12 12B" {
					t.Fatalf("log format fail")
				}
			},
		}
		m := NewLogger(config)
		req := httptest.NewRequest("GET", "https://aslant.site/?a=1&b=2", nil)
		req.Header.Set("Referer", "https://aslant.site/")
		req.Header.Set("User-Agent", "test-agent")
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.BodyBuffer = bytes.NewBufferString("response-body")
		c.RequestBody = []byte("request-body")
		c.StatusCode = 200
		c.Next = func() error {
			return nil
		}
		err := m(c)
		assert.Nil(err)
	})

	t.Run("latency", func(t *testing.T) {
		config := LoggerConfig{
			Format: "{latency} {latency-ms}",
			OnLog: func(log string, _ *elton.Context) {
				if len(strings.Split(log, " ")) != 2 {
					t.Fatalf("get latency fail")
				}
			},
		}
		m := NewLogger(config)
		req := httptest.NewRequest("GET", "https://aslant.iste/?a=1&b=2", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			time.Sleep(time.Second)
			return nil
		}
		err := m(c)
		assert.Nil(err)
	})

	t.Run("when", func(t *testing.T) {
		config := LoggerConfig{
			Format: "{when}  {when-iso}  {when-utc-iso}  {when-unix}  {when-iso-ms}  {when-utc-iso-ms}",
			OnLog: func(log string, _ *elton.Context) {
				if len(strings.Split(log, "  ")) != 6 {
					t.Fatalf("get when fail")
				}
			},
		}
		m := NewLogger(config)
		req := httptest.NewRequest("GET", "https://aslant.iste/?a=1&b=2", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		err := m(c)
		assert.Nil(err)
	})

	t.Run("cookie", func(t *testing.T) {
		config := LoggerConfig{
			Format: "{~jt}",
			OnLog: func(log string, _ *elton.Context) {
				if log != "abc" {
					t.Fatalf("get cookie value fail")
				}
			},
		}
		m := NewLogger(config)
		req := httptest.NewRequest("GET", "https://aslant.iste/?a=1&b=2", nil)
		req.AddCookie(&http.Cookie{
			Name:  "jt",
			Value: "abc",
		})
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		err := m(c)
		assert.Nil(err)
	})

	t.Run("header", func(t *testing.T) {
		config := LoggerConfig{
			Format: "{>X-Token} {<X-Response-Id} place-holder",
			OnLog: func(log string, _ *elton.Context) {
				if log != "abc def place-holder" {
					t.Fatalf("get header value fail")
				}
			},
		}
		m := NewLogger(config)
		req := httptest.NewRequest("GET", "https://aslant.iste/?a=1&b=2", nil)
		req.Header.Set("X-Token", "abc")
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.SetHeader("X-Response-Id", "def")
		c.Next = func() error {
			return nil
		}
		err := m(c)
		assert.Nil(err)
	})

	t.Run("get log function", func(t *testing.T) {
		layout := "{host} {remote} {real-ip} {method} {path} {proto} {query} {scheme} {uri} {referer} {userAgent} {size} {size-human} {status} {payload-size} {payload-size-human}"
		fn := GenerateLog(layout)
		req := httptest.NewRequest("GET", "https://aslant.site/?a=1&b=2", nil)
		req.Header.Set("Referer", "https://aslant.site/")
		req.Header.Set("User-Agent", "test-agent")
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.BodyBuffer = bytes.NewBufferString("response-body")
		c.RequestBody = []byte("request-body")
		c.StatusCode = 200
		startedAt := time.Now()
		assert.Equal("aslant.site 192.0.2.1:1234 192.0.2.1 GET / HTTP/1.1 a=1&b=2 HTTPS https://aslant.site/?a=1&b=2 https://aslant.site/ test-agent 13 13B 200 12 12B", fn(c, startedAt))
	})

}
