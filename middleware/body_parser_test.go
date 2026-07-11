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
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton/v2"
	"github.com/vicanso/hes"
)

type (
	errReadCloser struct {
		customErr error
	}
)

// Read read function
func (er *errReadCloser) Read(p []byte) (n int, err error) {
	return 0, er.customErr
}

// Close close function
func (er *errReadCloser) Close() error {
	return nil
}

// NewErrorReadCloser create an read error
func NewErrorReadCloser(err error) io.ReadCloser {
	r := &errReadCloser{
		customErr: err,
	}
	return r
}

func TestGzipDecoder(t *testing.T) {
	gzipDecoder := NewGzipDecoder()
	assert := assert.New(t)
	originalBuf := []byte("abcdabcdabcd")
	var b bytes.Buffer
	w, _ := gzip.NewWriterLevel(&b, 9)
	_, err := w.Write(originalBuf)
	assert.Nil(err)
	_ = w.Close()

	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	assert.False(gzipDecoder.Validate(c))

	c.SetRequestHeader(elton.HeaderContentEncoding, elton.Gzip)
	assert.True(gzipDecoder.Validate(c))

	tests := []struct {
		newContext func() *elton.Context
		data       []byte
		err        error
		result     []byte
	}{
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
				c.SetRequestHeader(elton.HeaderContentEncoding, "gzip")
				c.SetRequestHeader(elton.HeaderContentLength, "123")
				return c
			},
			data:   b.Bytes(),
			result: originalBuf,
		},
		// invalid gzip data
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
				return c
			},
			data: []byte("ab"),
			err:  errors.New("unexpected EOF"),
		},
	}

	for _, tt := range tests {
		c := tt.newContext()
		result, err := gzipDecoder.Decode(c, tt.data)
		assert.Equal(tt.err, err)
		assert.Equal(tt.result, result)
		if err == nil {
			assert.Empty(c.GetRequestHeader(elton.HeaderContentEncoding))
			assert.Empty(c.GetRequestHeader(elton.HeaderContentLength))
		}
	}
}

func TestJSONDecoder(t *testing.T) {
	assert := assert.New(t)
	jsonDecoder := NewJSONDecoder()
	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	assert.False(jsonDecoder.Validate(c))
	c.SetRequestHeader(elton.HeaderContentType, elton.MIMEApplicationJSON)
	assert.True(jsonDecoder.Validate(c))

	tests := []struct {
		data   []byte
		err    error
		result []byte
	}{
		{
			data:   []byte(`{"a": 1}`),
			result: []byte(`{"a": 1}`),
		},
		// empty data
		{
			data: []byte(""),
		},
		// invalid json
		{
			data: []byte("{"),
			err:  ErrInvalidJSON,
		},
		// invalid json
		{
			data: []byte("abcd"),
			err:  ErrInvalidJSON,
		},
		// invalid json
		{
			data: []byte("{abcd"),
			err:  ErrInvalidJSON,
		},
		// invalid json
		{
			data: []byte("[abcd"),
			err:  ErrInvalidJSON,
		},
	}

	for _, tt := range tests {
		result, err := jsonDecoder.Decode(c, tt.data)
		assert.Equal(tt.err, err)
		assert.Equal(tt.result, result)
	}
}

func TestFormURLEncodedDecoder(t *testing.T) {
	formURLEncodedDecoder := NewFormURLEncodedDecoder()
	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	assert.False(t, formURLEncodedDecoder.Validate(c))
	c.SetRequestHeader(elton.HeaderContentType, "application/x-www-form-urlencoded; charset=utf-8")
	assert.True(t, formURLEncodedDecoder.Validate(c))

	tests := []struct {
		name   string
		data   []byte
		err    error
		result []byte
		// size 用于 map 无序导致字段顺序不定时只校验长度
		size int
		// contains 校验 JSON 中必须出现的子串（转义/防注入）
		contains []string
		// notContains 校验不得出现的子串
		notContains []string
	}{
		{
			name: "simple pair",
			data: []byte("a=1&b=2"),
			// 格式化后的顺序有可能不一致，因此直接校验长度
			size: 17,
		},
		{
			name:   "multi value",
			data:   []byte("a=1&a=2"),
			result: []byte(`{"a":["1","2"]}`),
		},
		{
			name: "escape quotes and backslash",
			data: []byte(`a=1"2&b=x\y`),
			contains: []string{
				`"a":"1\"2"`,
				`"b":"x\\y"`,
			},
		},
		{
			name: "reject field injection via value",
			// 旧实现手工拼接会生成: {"a":"1","b":"","x":1"} 一类非法/注入 JSON
			data: []byte(`a=1&b=","x":1`),
			contains: []string{
				`"b":"\",\"x\":1"`,
			},
			notContains: []string{
				`"x":1`,
				`"x":"1"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formURLEncodedDecoder.Decode(c, tt.data)
			assert.Equal(t, tt.err, err)
			if tt.result != nil {
				assert.Equal(t, string(tt.result), string(result))
			}
			if tt.size != 0 {
				assert.Equal(t, tt.size, len(result))
			}
			for _, s := range tt.contains {
				assert.Contains(t, string(result), s)
			}
			for _, s := range tt.notContains {
				assert.NotContains(t, string(result), s)
			}
			// 结果必须是合法 JSON object
			if err == nil && len(result) > 0 {
				var m map[string]any
				assert.NoError(t, json.Unmarshal(result, &m))
			}
		})
	}
}

type testReadCloser struct {
	data *bytes.Buffer
}

func (trc *testReadCloser) Read(p []byte) (n int, err error) {
	return trc.data.Read(p)
}

func (trc *testReadCloser) Close() error {
	return nil
}

func TestMaxBytesReader(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		reader *testReadCloser
		max    int64
		err    error
		result []byte
	}{
		{
			reader: &testReadCloser{
				data: bytes.NewBufferString("abcd"),
			},
			max:    1,
			err:    errors.New("request body is too large, it should be <= 1"),
			result: []byte("a"),
		},
		{
			reader: &testReadCloser{
				data: bytes.NewBufferString("abcd"),
			},
			max:    100,
			result: []byte("abcd"),
		},
	}
	for _, tt := range tests {
		r := MaxBytesReader(tt.reader, tt.max)
		result, err := io.ReadAll(r)
		assert.Equal(tt.err, err)
		assert.Equal(tt.result, result)
	}
}

type testDecoder struct{}

func (td *testDecoder) Validate(c *elton.Context) bool {
	return c.GetRequestHeader(elton.HeaderContentType) == "application/json;charset=base64"
}
func (td *testDecoder) Decode(c *elton.Context, originalData []byte) (data []byte, err error) {
	return base64.RawStdEncoding.DecodeString(string(originalData))
}

func TestBodyParserMiddleware(t *testing.T) {
	assert := assert.New(t)
	skipErr := errors.New("skip error")
	// next直接返回skip error，用于判断是否执行了next
	next := func() error {
		return skipErr
	}
	defaultBodyParser := NewDefaultBodyParser()
	beforeDecodeCount := int32(0)
	onBeforeDecode := func(_ *elton.Context) error {
		atomic.AddInt32(&beforeDecodeCount, 1)
		return nil
	}

	formConf := BodyParserConfig{
		ContentTypeValidate: DefaultJSONAndFormContentTypeValidate,
		OnBeforeDecode:      onBeforeDecode,
	}
	formConf.AddDecoder(NewFormURLEncodedDecoder())
	formParser := NewBodyParser(formConf)

	customConf := BodyParserConfig{
		OnBeforeDecode: onBeforeDecode,
	}
	customConf.AddDecoder(&testDecoder{})
	customParser := NewBodyParser(customConf)

	readErr := errors.New("abc")
	tests := []struct {
		newContext  func() *elton.Context
		fn          elton.Handler
		err         error
		requestBody []byte
	}{
		// read error
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("POST", "/", NewErrorReadCloser(readErr))
				req.Header.Set(elton.HeaderContentType, "application/json")
				c := elton.NewContext(nil, req)
				return c
			},
			fn: defaultBodyParser,
			err: &hes.Error{
				Exception:  true,
				StatusCode: http.StatusInternalServerError,
				Message:    readErr.Error(),
				Category:   ErrBodyParserCategory,
				Cause:      readErr,
			},
		},
		// over limit
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("POST", "/", strings.NewReader("abc"))
				req.Header.Set(elton.HeaderContentType, "application/json")
				c := elton.NewContext(nil, req)
				return c
			},
			fn: NewBodyParser(BodyParserConfig{
				Limit: 1,
			}),
			err: &hes.Error{
				Exception:  true,
				StatusCode: http.StatusInternalServerError,
				Message:    "request body is too large, it should be <= 1",
				Category:   ErrBodyParserCategory,
				Cause:      errors.New("request body is too large, it should be <= 1"),
			},
		},
		// committed: true
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(nil, nil)
				c.Committed = true
				c.Next = next
				return c
			},
			fn:  defaultBodyParser,
			err: skipErr,
		},
		// request body is not nil
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(nil, nil)
				c.RequestBody = []byte("abc")
				c.Next = next
				return c
			},
			requestBody: []byte("abc"),
			fn:          defaultBodyParser,
			err:         skipErr,
		},
		// content type is not json
		{
			newContext: func() *elton.Context {
				// 未设置content type
				c := elton.NewContext(nil, httptest.NewRequest("POST", "/", nil))
				c.Next = next
				return c
			},
			fn:  defaultBodyParser,
			err: skipErr,
		},
		// method is get(pass)
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(nil, httptest.NewRequest("GET", "/", nil))
				c.Request.Header.Set(elton.HeaderContentType, "application/json")
				c.Next = next
				return c
			},
			fn:  defaultBodyParser,
			err: skipErr,
		},
		// json
		{
			newContext: func() *elton.Context {
				body := `{"name": "tree.xie"}`
				req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader(body))
				req.Header.Set(elton.HeaderContentType, "application/json")
				req.Header.Set(elton.HeaderContentLength, strconv.Itoa(len(body)))
				c := elton.NewContext(nil, req)
				c.Next = next
				return c
			},
			fn:          defaultBodyParser,
			err:         skipErr,
			requestBody: []byte(`{"name": "tree.xie"}`),
		},
		// json + gzip
		{
			newContext: func() *elton.Context {
				originalBuf := []byte(`{"name": "tree.xie"}`)
				var b bytes.Buffer
				w, _ := gzip.NewWriterLevel(&b, 9)
				_, err := w.Write(originalBuf)
				assert.Nil(err)
				_ = w.Close()

				req := httptest.NewRequest("POST", "https://aslant.site/", bytes.NewReader(b.Bytes()))
				req.Header.Set(elton.HeaderContentType, "application/json")
				req.Header.Set(elton.HeaderContentEncoding, "gzip")
				c := elton.NewContext(nil, req)
				c.Next = next
				return c
			},
			fn:          defaultBodyParser,
			requestBody: []byte(`{"name": "tree.xie"}`),
			err:         skipErr,
		},
		// form
		{
			newContext: func() *elton.Context {
				body := `type=1&type=2`
				req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader(body))
				req.Header.Set(elton.HeaderContentType, "application/x-www-form-urlencoded")
				c := elton.NewContext(nil, req)
				c.Next = next
				return c
			},
			fn:          formParser,
			requestBody: []byte(`{"type":["1","2"]}`),
			err:         skipErr,
		},
		// custom decoder
		{
			newContext: func() *elton.Context {
				body := `{"name": "tree.xie"}`
				b64 := base64.RawStdEncoding.EncodeToString([]byte(body))
				req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader(b64))
				req.Header.Set(elton.HeaderContentType, "application/json;charset=base64")
				c := elton.NewContext(nil, req)
				c.Next = next
				return c
			},
			fn:          customParser,
			requestBody: []byte(`{"name": "tree.xie"}`),
			err:         skipErr,
		},
	}
	for _, tt := range tests {
		c := tt.newContext()
		err := tt.fn(c)
		assert.Equal(tt.err, err)
		assert.Equal(tt.requestBody, c.RequestBody)
	}
	assert.Equal(2, int(beforeDecodeCount))
}

func TestRequestBodyReader(t *testing.T) {
	assert := assert.New(t)

	rr := requestBodyReader{
		limit:      5 * 1024,
		bufferPool: elton.NewBufferPool(1024),
	}
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(randomString(2048)))
	c := elton.NewContext(httptest.NewRecorder(), req)
	buf, err := rr.ReadAll(c)
	assert.Nil(err)
	assert.Equal(2732, len(buf))

	req = httptest.NewRequest("POST", "/", bytes.NewBufferString(randomString(3000)))
	req.Header.Set(elton.HeaderContentLength, strconv.Itoa(3000))
	c = elton.NewContext(httptest.NewRecorder(), req)
	buf, err = rr.ReadAll(c)
	assert.Nil(err)
	assert.Equal(4000, len(buf))
}

// TestRequestBodyReaderContentLengthCap 确保恶意过大的 Content-Length
// 不会被用于预分配超过 limit 的切片。
func TestRequestBodyReaderContentLengthCap(t *testing.T) {
	assert := assert.New(t)
	limit := 64
	rr := requestBodyReader{
		limit:      limit,
		bufferPool: elton.NewBufferPool(32),
	}
	body := []byte(`{"ok":true}`)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	// 声明远大于实际 body 与 limit 的 CL
	req.Header.Set(elton.HeaderContentLength, strconv.Itoa(1<<30))
	c := elton.NewContext(httptest.NewRecorder(), req)
	buf, err := rr.ReadAll(c)
	// 实际 body 小于 limit，应成功读出
	assert.Nil(err)
	assert.Equal(body, buf)

	// 非法 / 非正 Content-Length 走 buffer 路径，不 panic
	req = httptest.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set(elton.HeaderContentLength, "not-a-number")
	c = elton.NewContext(httptest.NewRecorder(), req)
	buf, err = rr.ReadAll(c)
	assert.Nil(err)
	assert.Equal(body, buf)

	req = httptest.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set(elton.HeaderContentLength, "-1")
	c = elton.NewContext(httptest.NewRecorder(), req)
	buf, err = rr.ReadAll(c)
	assert.Nil(err)
	assert.Equal(body, buf)
}

func BenchmarkBodyParserBufferPool(b *testing.B) {
	size := 5 * 1024
	fn := NewBodyParser(BodyParserConfig{
		InitCap: size,
	})
	for i := 0; i < b.N; i++ {
		bodySize := rand.Intn(size)
		body := bytes.NewReader([]byte(randomString(bodySize)))
		req := httptest.NewRequest("POST", "/", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		realBodySize := bodySize / 3 * 4
		if bodySize%3 != 0 {
			realBodySize += 4
		}
		if err != nil || len(c.RequestBody) != realBodySize {
			panic("get request body fail")
		}
	}
}
