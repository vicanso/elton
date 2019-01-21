package middleware

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vicanso/cod"
)

const (
	staticPath = "/local"
)

type MockStaticFile struct {
}
type MockFileStat struct{}

func (m *MockStaticFile) Exists(file string) bool {
	if strings.HasSuffix(file, "notfound.html") {
		return false
	}
	return true
}

func (m *MockStaticFile) Get(file string) ([]byte, error) {
	if file == staticPath+"/index.html" {
		return []byte("<html>xxx</html>"), nil
	}
	return []byte("abcd"), nil
}

func (m *MockStaticFile) Stat(file string) os.FileInfo {
	return &MockFileStat{}
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
	return time.Now()
}

func (mf *MockFileStat) IsDir() bool {
	return false
}

func (mf *MockFileStat) Sys() interface{} {
	return nil
}

func TestFS(t *testing.T) {
	file := os.Args[0]
	fs := FS{}
	if !fs.Exists(file) {
		t.Fatalf("file should be exists")
	}
	fileInfo := fs.Stat(file)
	if fileInfo == nil {
		t.Fatalf("stat file fail")
	}

	buf, err := fs.Get(file)
	if err != nil || len(buf) == 0 {
		t.Fatalf("get file fail, %v", err)
	}

	t.Run("out of path", func(t *testing.T) {
		tfs := FS{
			Path: "/a",
		}
		if tfs.Stat("/b") != nil {
			t.Fatalf("out of path should return nil stat")
		}
		if tfs.Exists("/b") {
			t.Fatalf("file is not exists")
		}
		_, err := tfs.Get("/b")
		if err != ErrOutOfPath {
			t.Fatalf("should return out of path")
		}
	})
}
func TestStaticServe(t *testing.T) {
	staticFile := &MockStaticFile{}
	t.Run("not allow query string", func(t *testing.T) {
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:            staticPath,
			DenyQueryString: true,
		})
		req := httptest.NewRequest("GET", "/index.html?a=1", nil)
		c := cod.NewContext(nil, req)
		err := fn(c)
		if err != ErrNotAllowQueryString {
			t.Fatalf("should return not allow query string error")
		}
	})

	t.Run("pass mount", func(t *testing.T) {
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:  staticPath,
			Mount: "/static",
		})
		res := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/index.html", nil)
		c := cod.NewContext(res, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		if err != nil {
			t.Fatalf("pass mount fail, %v", err)
		}
		if !done || c.StatusCode != 0 {
			t.Fatalf("pass mount fail")
		}
	})

	t.Run("not found return error", func(t *testing.T) {
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path: staticPath,
		})
		req := httptest.NewRequest("GET", "/notfound.html", nil)
		c := cod.NewContext(nil, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		if err != ErrNotFound {
			t.Fatalf("should return not found error")
		}
	})

	t.Run("not found pass to next", func(t *testing.T) {
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:         staticPath,
			NotFoundNext: true,
		})
		req := httptest.NewRequest("GET", "/notfound.html", nil)
		c := cod.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		if err != nil || !done {
			t.Fatalf("not found pass fail, %v", err)
		}
	})

	t.Run("not compresss", func(t *testing.T) {
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:  staticPath,
			Mount: "/static",
		})
		req := httptest.NewRequest("GET", "/static/banner.jpg", nil)
		res := httptest.NewRecorder()
		c := cod.NewContext(res, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		if err != nil || c.GetHeader(cod.HeaderContentEncoding) == "gzip" {
			t.Fatalf("serve image fail, %v", err)
		}
	})

	t.Run("get index.html", func(t *testing.T) {
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:  staticPath,
			Mount: "/static",
		})
		req := httptest.NewRequest("GET", "/static/index.html?a=1", nil)
		res := httptest.NewRecorder()
		c := cod.NewContext(res, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		if err != nil {
			t.Fatalf("serve index.html fail, %v", err)
		}
		if c.GetHeader(cod.HeaderETag) != `"10-FKjW3bSjaJvr_QYzQcHNFRn-rxc="` ||
			c.GetHeader(cod.HeaderLastModified) == "" ||
			c.GetHeader("Content-Type") != "text/html; charset=utf-8" {
			t.Fatalf("set header fail")
		}
		if c.BodyBuffer.Len() != 16 {
			t.Fatalf("response body fail")
		}
	})

	t.Run("set custom header", func(t *testing.T) {
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path: staticPath,
			Header: map[string]string{
				"X-IDC": "GZ",
			},
		})
		req := httptest.NewRequest("GET", "/index.html", nil)
		res := httptest.NewRecorder()
		c := cod.NewContext(res, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		if err != nil || c.GetHeader("X-IDC") != "GZ" {
			t.Fatalf("set custom header fail, %v", err)
		}
	})

	t.Run("set (s)max-age", func(t *testing.T) {
		fn := NewStaticServe(staticFile, StaticServeConfig{
			Path:    staticPath,
			MaxAge:  24 * 3600,
			SMaxAge: 300,
		})
		req := httptest.NewRequest("GET", "/index.html", nil)
		res := httptest.NewRecorder()
		c := cod.NewContext(res, req)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		if err != nil || c.GetHeader(cod.HeaderCacheControl) != "public, max-age=86400, s-maxage=300" {
			t.Fatalf("set max age header fail, %v", err)
		}
	})
}
