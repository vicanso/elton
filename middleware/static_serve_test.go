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

func TestFSOutOfPath(t *testing.T) {
	assert := assert.New(t)
	fs := FS{}

	assert.Nil(fs.Stat("/b"), "out of path should return nil stat")
	assert.False(fs.Exists("/b"), "file should be not exists")
}
func TestFS(t *testing.T) {
	assert := assert.New(t)
	file := os.Args[0]
	fs := FS{}
	assert.NotNil(NewDefaultStaticServe(StaticServeConfig{}))
	assert.True(fs.Exists(file), "file should be exists")

	fileInfo := fs.Stat(file)
	assert.NotNil(fileInfo, "stat of file shouldn't be nil")

	buf, err := fs.Get(file)
	assert.Nil(err)
	assert.NotEmpty(buf)
}

func TestStaticServe(t *testing.T) {
	assert := assert.New(t)
	defaultStatic := NewStaticServe(&MockStaticFile{}, StaticServeConfig{
		Path:             staticPath,
		EnableStrongETag: true,
		DenyQueryString:  true,
		DenyDot:          true,
		Header: map[string]string{
			"X-IDC": "GZ",
		},
		MaxAge:    24 * time.Hour,
		SMaxAge:   5 * time.Minute,
		Immutable: true,
	})

	tests := []struct {
		newContext   func() *elton.Context
		err          error
		eTag         string
		contentType  string
		idc          string
		cacheControl string
	}{
		// deny query string
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/index.html?a=1", nil)
				c := elton.NewContext(httptest.NewRecorder(), req)
				return c
			},
			err: ErrStaticServeNotAllowQueryString,
		},
		// not allow dot file
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/.index.html", nil)
				c := elton.NewContext(httptest.NewRecorder(), req)
				return c
			},
			err: ErrStaticServeNotAllowAccessDot,
		},
		// not found
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/notfound.html", nil)
				c := elton.NewContext(httptest.NewRecorder(), req)
				c.Next = func() error {
					return nil
				}
				return c
			},
			err: ErrStaticServeNotFound,
		},
		// out of path
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/index.html", nil)
				req.URL.Path = "../../index.html"
				res := httptest.NewRecorder()
				c := elton.NewContext(res, req)
				c.Next = func() error {
					return nil
				}
				return c
			},
			err: ErrStaticServeOutOfPath,
		},
		// get file fail
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/error", nil)
				res := httptest.NewRecorder()
				c := elton.NewContext(res, req)
				c.Next = func() error {
					return nil
				}
				return c
			},
			err: errors.New("category=elton-static-serve, message=abcd"),
		},
		// image
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/banner.jpg", nil)
				res := httptest.NewRecorder()
				c := elton.NewContext(res, req)
				c.Next = func() error {
					return nil
				}
				return c
			},
			eTag:         `"a-1oFGwuX-Q3qfLHqK_7iCcc_0YYI="`,
			contentType:  "image/jpeg",
			idc:          "GZ",
			cacheControl: "public, max-age=86400, s-maxage=300, immutable",
		},
		// index html
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/index.html", nil)
				res := httptest.NewRecorder()
				c := elton.NewContext(res, req)
				c.Next = func() error {
					return nil
				}
				return c
			},
			eTag:         `"10-FKjW3bSjaJvr_QYzQcHNFRn-rxc="`,
			contentType:  "text/html; charset=utf-8",
			idc:          "GZ",
			cacheControl: "public, max-age=86400, s-maxage=300, immutable",
		},
	}
	for _, tt := range tests {
		c := tt.newContext()
		err := defaultStatic(c)
		if err != nil || tt.err != nil {
			assert.Equal(tt.err.Error(), err.Error())
		}
		assert.Equal(tt.eTag, c.GetHeader(elton.HeaderETag))
		assert.Equal(tt.contentType, c.GetHeader(elton.HeaderContentType))
		assert.Equal(tt.idc, c.GetHeader("X-IDC"))
		assert.Equal(tt.cacheControl, c.GetHeader(elton.HeaderCacheControl))
	}
}
