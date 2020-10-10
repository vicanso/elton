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
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

const (
	staticPath = "/local"
)

type MockStaticFile struct {
}
type MockFileStat struct{}

func (m *MockStaticFile) Exists(file string) bool {
	return !strings.HasSuffix(file, "notfound.html")
}

func (m *MockStaticFile) Get(file string) ([]byte, error) {
	if file == staticPath+"/error" {
		return nil, errors.New("abcd")
	}
	if file == staticPath+"/index.html" {
		return []byte("<html>xxx</html>"), nil
	}
	if file == staticPath+"/banner.jpg" {
		return []byte("image data"), nil
	}
	return []byte("abcd"), nil
}

func (m *MockStaticFile) Stat(file string) os.FileInfo {
	return &MockFileStat{}
}

func (m *MockStaticFile) NewReader(file string) (io.Reader, error) {
	buf, err := m.Get(file)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf), nil
}

func (mf *MockFileStat) Name() string {
	return "file"
}

func (mf *MockFileStat) Size() int64 {
	return 1024
}

func (mf *MockFileStat) Mode() os.FileMode {
	return os.ModeAppend
}

func (mf *MockFileStat) ModTime() time.Time {
	t, _ := time.Parse(time.RFC3339, "2019-06-08T02:17:54Z")
	return t
}

func (mf *MockFileStat) IsDir() bool {
	return false
}

func (mf *MockFileStat) Sys() interface{} {
	return nil
}

func TestGenerateETag(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(`"0-2jmj7l5rSw0yVb_vlWAYkK_YBwk="`, generateETag([]byte("")))
	assert.Equal(`"3-qZk-NkcGgWq6PiVxeFDCbJzQ2J0="`, generateETag([]byte("abc")))
}

func TestFS(t *testing.T) {
	file := os.Args[0]
	fs := FS{}
	t.Run("normal", func(t *testing.T) {
		assert := assert.New(t)
		assert.NotNil(NewDefaultStaticServe(StaticServeConfig{}))
		assert.True(fs.Exists(file), "file should be exists")

		fileInfo := fs.Stat(file)
		assert.NotNil(fileInfo, "stat of file shouldn't be nil")

		buf, err := fs.Get(file)
		assert.Nil(err)
		assert.NotEmpty(buf)
	})

	t.Run("out of path", func(t *testing.T) {
		assert := assert.New(t)
		tfs := FS{}

		assert.Nil(tfs.Stat("/b"), "out of path should return nil stat")
		assert.False(tfs.Exists("/b"), "file should be not exists")
	})
}
func TestStaticServe(t *testing.T) {
	staticFile := &MockStaticFile{}
	t.Run("not allow query string", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:            staticPath,
			DenyQueryString: true,
		})
		req := httptest.NewRequest("GET", "/index.html?a=1", nil)
		c := elton.NewContext(nil, req)
		err := fn(c)
		assert.Equal(ErrStaticServeNotAllowQueryString, err, "should return not allow query string error")
	})

	t.Run("not allow dot file", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:    staticPath,
			DenyDot: true,
		})
		req := httptest.NewRequest("GET", "/.index.html", nil)
		c := elton.NewContext(nil, req)
		err := fn(c)
		assert.Equal(ErrStaticServeNotAllowAccessDot, err, "should return not allow dot error")
	})

	t.Run("not found return error", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path: staticPath,
		})
		req := httptest.NewRequest("GET", "/notfound.html", nil)
		c := elton.NewContext(nil, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.Equal(ErrStaticServeNotFound, err, "should return not found error")
	})

	t.Run("not found pass to next", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:         staticPath,
			NotFoundNext: true,
		})
		req := httptest.NewRequest("GET", "/notfound.html", nil)
		c := elton.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
	})

	t.Run("not compresss", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:             staticPath,
			EnableStrongETag: true,
		})
		req := httptest.NewRequest("GET", "/banner.jpg", nil)
		res := httptest.NewRecorder()
		c := elton.NewContext(res, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.NotEqual("gzip", c.GetHeader(elton.HeaderContentEncoding))
		assert.Equal(`"a-1oFGwuX-Q3qfLHqK_7iCcc_0YYI="`, c.GetHeader(elton.HeaderETag))
	})

	t.Run("get index.html", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path: staticPath,
		})
		req := httptest.NewRequest("GET", "/index.html?a=1", nil)
		res := httptest.NewRecorder()
		c := elton.NewContext(res, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.Nil(err, "serve index.html fail")

		assert.Equal(`W/"400-5cfb1ad2"`, c.GetHeader(elton.HeaderETag), "generate etag fail")
		assert.NotEmpty(c.GetHeader(elton.HeaderLastModified), "last modified shouldn't be empty")
		assert.Equal("text/html; charset=utf-8", c.GetHeader("Content-Type"))
		assert.True(c.IsReaderBody())
	})

	t.Run("set custom header", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path: staticPath,
			Header: map[string]string{
				"X-IDC": "GZ",
			},
		})
		req := httptest.NewRequest("GET", "/index.html", nil)
		res := httptest.NewRecorder()
		c := elton.NewContext(res, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.Equal("GZ", c.GetHeader("X-IDC"), "set custom header fail")
	})

	t.Run("set (s)max-age", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:    staticPath,
			MaxAge:  24 * time.Hour,
			SMaxAge: 5 * time.Minute,
		})
		req := httptest.NewRequest("GET", "/index.html", nil)
		res := httptest.NewRecorder()
		c := elton.NewContext(res, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.Equal("public, max-age=86400, s-maxage=300", c.GetHeader(elton.HeaderCacheControl), "set max age header fail")
	})

	t.Run("out of path", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:    staticPath,
			MaxAge:  24 * 3600,
			SMaxAge: 300,
		})
		req := httptest.NewRequest("GET", "/index.html", nil)
		req.URL.Path = "../../index.html"
		res := httptest.NewRecorder()
		c := elton.NewContext(res, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.Equal("category=elton-static-serve, message=out of path", err.Error(), "out of path should return error")
	})

	t.Run("get file error", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:    staticPath,
			MaxAge:  24 * 3600,
			SMaxAge: 300,
		})
		req := httptest.NewRequest("GET", "/error", nil)
		res := httptest.NewRecorder()
		c := elton.NewContext(res, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.Equal("category=elton-static-serve, message=abcd", err.Error(), "get file fail should return error")
	})
}
