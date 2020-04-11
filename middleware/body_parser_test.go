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
	"io"
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

func TestBodyParser(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		assert := assert.New(t)
		bodyParser := NewBodyParser(BodyParserConfig{
			Skipper: func(c *elton.Context) bool {
				return true
			},
		})

		body := `{"name": "tree.xie"}`
		req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader(body))
		req.Header.Set(elton.HeaderContentType, "application/json")
		c := elton.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := bodyParser(c)
		assert.Nil(err)
		assert.True(done)
		assert.Equal(len(c.RequestBody), 0)
	})

	t.Run("request body content type is not json", func(t *testing.T) {
		assert := assert.New(t)
		bodyParser := NewDefaultBodyParser()

		body := `<xml>xxx</xml>`
		req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader(body))
		req.Header.Set(elton.HeaderContentType, "application/xml")
		c := elton.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := bodyParser(c)

		assert.Nil(err)
		assert.True(done)
		assert.Nil(c.RequestBody)
	})

	t.Run("request body is not nil", func(t *testing.T) {
		assert := assert.New(t)
		bodyParser := NewDefaultBodyParser()

		body := `{"name": "tree.xie"}`
		req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader(body))
		req.Header.Set(elton.HeaderContentType, "application/json")
		c := elton.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		c.RequestBody = []byte("a")
		err := bodyParser(c)

		assert.Nil(err)
		assert.True(done)
		assert.Equal(c.RequestBody, []byte("a"))
	})

	t.Run("pass method", func(t *testing.T) {
		assert := assert.New(t)
		bodyParser := NewBodyParser(BodyParserConfig{})
		req := httptest.NewRequest("GET", "https://aslant.site/", nil)
		c := elton.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := bodyParser(c)
		assert.Nil(err)
		assert.True(done)
	})

	t.Run("pass content type not json", func(t *testing.T) {
		assert := assert.New(t)
		bodyParser := NewBodyParser(BodyParserConfig{})
		req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader("abc"))
		c := elton.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := bodyParser(c)
		assert.Nil(err)
		assert.True(done)
	})

	t.Run("read body fail", func(t *testing.T) {
		assert := assert.New(t)
		bodyParser := NewBodyParser(BodyParserConfig{})
		req := httptest.NewRequest("POST", "https://aslant.site/", NewErrorReadCloser(hes.New("abc")))
		req.Header.Set(elton.HeaderContentType, "application/json")
		c := elton.NewContext(nil, req)
		err := bodyParser(c)
		assert.NotNil(err)
		assert.Equal(err.Error(), "category=elton-body-parser, message=message=abc")
	})

	t.Run("body over limit size", func(t *testing.T) {
		assert := assert.New(t)
		bodyParser := NewBodyParser(BodyParserConfig{
			Limit: 1,
		})
		req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader("abc"))
		req.Header.Set(elton.HeaderContentType, "application/json")
		c := elton.NewContext(nil, req)
		err := bodyParser(c)
		assert.NotNil(err)
		assert.Equal(err.Error(), "category=elton-body-parser, message=request body is too large, it should be <= 1")
	})

	t.Run("parse json success", func(t *testing.T) {
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
	})

	t.Run("parse json(gzip) success", func(t *testing.T) {
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
	})

	t.Run("decode data success", func(t *testing.T) {
		assert := assert.New(t)
		conf := BodyParserConfig{}
		conf.AddDecoder(&BodyDecoder{
			Validate: func(c *elton.Context) bool {
				return c.GetRequestHeader(elton.HeaderContentType) == "application/json;charset=base64"
			},
			Decode: func(c *elton.Context, originalData []byte) (data []byte, err error) {
				return base64.RawStdEncoding.DecodeString(string(originalData))
			},
		})

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
	})

	t.Run("parse form url encoded success", func(t *testing.T) {
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
	})
}
