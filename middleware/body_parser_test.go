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
	"errors"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
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
	w.Close()

	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	assert.False(gzipDecoder.Validate(c))

	c.SetRequestHeader(elton.HeaderContentEncoding, elton.Gzip)
	assert.True(gzipDecoder.Validate(c))
	buf, err := gzipDecoder.Decode(c, b.Bytes())
	assert.Nil(err)
	assert.Equal(originalBuf, buf)

	_, err = gzipDecoder.Decode(c, []byte("ab"))
	assert.NotNil(err)
}

func TestJSONDecoder(t *testing.T) {
	assert := assert.New(t)
	jsonDecoder := NewJSONDecoder()
	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	assert.False(jsonDecoder.Validate(c))
	c.SetRequestHeader(elton.HeaderContentType, elton.MIMEApplicationJSON)
	assert.True(jsonDecoder.Validate(c))

	buf := []byte(`{"a": 1}`)
	data, err := jsonDecoder.Decode(c, buf)
	assert.Nil(err)
	assert.Equal(buf, data)

	buf = []byte(``)
	data, err = jsonDecoder.Decode(c, buf)
	assert.Nil(err)
	assert.Nil(data)

	_, err = jsonDecoder.Decode(c, []byte("{"))
	assert.Equal(ErrInvalidJSON, err)

	_, err = jsonDecoder.Decode(c, []byte("abcd"))
	assert.Equal(ErrInvalidJSON, err)

	_, err = jsonDecoder.Decode(c, []byte("{abcd"))
	assert.Equal(ErrInvalidJSON, err)

	_, err = jsonDecoder.Decode(c, []byte("[abcd"))
	assert.Equal(ErrInvalidJSON, err)
}

func TestFormURLEncodedDecoder(t *testing.T) {
	assert := assert.New(t)
	formURLEncodedDecoder := NewFormURLEncodedDecoder()
	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	assert.False(formURLEncodedDecoder.Validate(c))
	c.SetRequestHeader(elton.HeaderContentType, "application/x-www-form-urlencoded; charset=UTF-8")
	assert.True(formURLEncodedDecoder.Validate(c))

	data, err := formURLEncodedDecoder.Decode(c, []byte("a=1&b=2"))
	assert.Nil(err)
	assert.Equal(17, len(data))
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
	buf := bytes.NewBufferString("abcd")
	trc := &testReadCloser{
		data: buf,
	}
	// 限制只能最大只能读取1字节，则出错
	r := MaxBytesReader(trc, 1)
	_, err := ioutil.ReadAll(r)
	assert.Equal("request body is too large, it should be <= 1", err.Error())

	buf = bytes.NewBufferString("abcd")
	result := buf.String()
	trc = &testReadCloser{
		data: buf,
	}
	// 限制最大100字节，则成功读取
	r = MaxBytesReader(trc, 100)
	data, err := ioutil.ReadAll(r)
	assert.Nil(err)
	assert.Equal(result, string(data))
}

type testDecoder struct{}

func (td *testDecoder) Validate(c *elton.Context) bool {
	return c.GetRequestHeader(elton.HeaderContentType) == "application/json;charset=base64"
}
func (td *testDecoder) Decode(c *elton.Context, originalData []byte) (data []byte, err error) {
	return base64.RawStdEncoding.DecodeString(string(originalData))
}

func TestBodyParserSkip(t *testing.T) {
	assert := assert.New(t)
	skipErr := errors.New("skip error")
	// next直接返回skip error，用于判断是否执行了next
	next := func() error {
		return skipErr
	}
	tests := []struct {
		newContext func() *elton.Context
	}{
		// commited: true
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(nil, nil)
				c.Committed = true
				c.Next = next
				return c
			},
		},
		// request body is not nil
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(nil, nil)
				c.RequestBody = []byte("abc")
				c.Next = next
				return c
			},
		},
		// content type is not json
		{
			newContext: func() *elton.Context {
				// 未设置content type
				c := elton.NewContext(nil, httptest.NewRequest("POST", "/", nil))
				c.Next = next
				return c
			},
		},
		// method is get(pass)
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(nil, httptest.NewRequest("GET", "/", nil))
				c.Request.Header.Set(elton.HeaderContentType, "application/json")
				c.Next = next
				return c
			},
		},
	}
	bodyPraser := NewDefaultBodyParser()
	for _, tt := range tests {
		err := bodyPraser(tt.newContext())
		assert.Equal(skipErr, err)
	}
}

func TestBodyParserReadFail(t *testing.T) {
	// 读取数据失败
	assert := assert.New(t)
	bodyParser := NewBodyParser(BodyParserConfig{})
	req := httptest.NewRequest("POST", "/", NewErrorReadCloser(hes.New("abc")))
	req.Header.Set(elton.HeaderContentType, "application/json")
	c := elton.NewContext(nil, req)
	err := bodyParser(c)
	assert.NotNil(err)
	assert.Equal("category=elton-body-parser, message=message=abc", err.Error())
}

func TestBodyParserOverLimit(t *testing.T) {
	assert := assert.New(t)
	bodyParser := NewBodyParser(BodyParserConfig{
		Limit: 1,
	})
	req := httptest.NewRequest("POST", "/", strings.NewReader("abc"))
	req.Header.Set(elton.HeaderContentType, "application/json")
	c := elton.NewContext(nil, req)
	err := bodyParser(c)
	assert.NotNil(err)
	assert.Equal("category=elton-body-parser, message=request body is too large, it should be <= 1", err.Error())
}

func TestBodyParserJSON(t *testing.T) {
	assert := assert.New(t)
	conf := BodyParserConfig{}
	conf.AddDecoder(NewJSONDecoder())
	bodyParser := NewBodyParser(conf)
	body := `{"name": "tree.xie"}`
	req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader(body))
	req.Header.Set(elton.HeaderContentType, "application/json")
	c := elton.NewContext(nil, req)
	done := false
	c.Next = func() error {
		done = true
		if string(c.RequestBody) != body {
			return hes.New("request body is invalid")
		}
		return nil
	}
	err := bodyParser(c)
	assert.Nil(err)
	assert.True(done)
}

func TestBodyParserJSONGzip(t *testing.T) {
	assert := assert.New(t)
	conf := BodyParserConfig{}
	conf.AddDecoder(NewGzipDecoder())
	conf.AddDecoder(NewJSONDecoder())
	bodyParser := NewBodyParser(conf)
	originalBuf := []byte(`{"name": "tree.xie"}`)
	var b bytes.Buffer
	w, _ := gzip.NewWriterLevel(&b, 9)
	_, err := w.Write(originalBuf)
	assert.Nil(err)
	w.Close()

	req := httptest.NewRequest("POST", "https://aslant.site/", bytes.NewReader(b.Bytes()))
	req.Header.Set(elton.HeaderContentType, "application/json")
	req.Header.Set(elton.HeaderContentEncoding, "gzip")
	c := elton.NewContext(nil, req)
	done := false
	c.Next = func() error {
		done = true
		if !bytes.Equal(c.RequestBody, originalBuf) {
			return hes.New("request body is invalid")
		}
		return nil
	}
	err = bodyParser(c)
	assert.Nil(err)
	assert.True(done)
}

func TestBodyParserFormURLEncoded(t *testing.T) {
	assert := assert.New(t)
	conf := BodyParserConfig{
		ContentTypeValidate: DefaultJSONAndFormContentTypeValidate,
	}
	conf.AddDecoder(NewFormURLEncodedDecoder())
	bodyParser := NewBodyParser(conf)
	body := `name=tree.xie&type=1&type=2`
	req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader(body))
	req.Header.Set(elton.HeaderContentType, "application/x-www-form-urlencoded")
	c := elton.NewContext(nil, req)
	done := false
	c.Next = func() error {
		done = true
		if len(c.RequestBody) != 36 {
			return hes.New("request body is invalid")
		}
		return nil
	}
	err := bodyParser(c)
	assert.Nil(err)
	assert.True(done)
}

func TestBodyParserTestDecoder(t *testing.T) {
	assert := assert.New(t)
	conf := BodyParserConfig{}
	conf.AddDecoder(&testDecoder{})

	bodyParser := NewBodyParser(conf)
	body := `{"name": "tree.xie"}`
	b64 := base64.RawStdEncoding.EncodeToString([]byte(body))
	req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader(b64))
	req.Header.Set(elton.HeaderContentType, "application/json;charset=base64")
	c := elton.NewContext(nil, req)
	done := false
	c.Next = func() error {
		done = true
		if string(c.RequestBody) != body {
			return hes.New("request body is invalid")
		}
		return nil
	}
	err := bodyParser(c)
	assert.Nil(err)
	assert.True(done)
}
