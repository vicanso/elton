package middleware

import (
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vicanso/cod"
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

func TestBodyParser(t *testing.T) {
	t.Run("pass method", func(t *testing.T) {
		bodyParser := NewBodyParser(BodyParserConfig{})
		req := httptest.NewRequest("GET", "https://aslant.site/", nil)
		c := cod.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := bodyParser(c)
		if err != nil {
			t.Fatalf("json parse fail, %v", err)
		}
		if !done {
			t.Fatalf("json parse fail")
		}
	})

	t.Run("pass content type not json", func(t *testing.T) {
		bodyParser := NewBodyParser(BodyParserConfig{})
		req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader("abc"))
		c := cod.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := bodyParser(c)
		if err != nil {
			t.Fatalf("body parse fail, %v", err)
		}
		if !done {
			t.Fatalf("body parse fail")
		}
	})

	t.Run("read body fail", func(t *testing.T) {
		bodyParser := NewBodyParser(BodyParserConfig{})
		req := httptest.NewRequest("POST", "https://aslant.site/", NewErrorReadCloser(errors.New("abc")))
		req.Header.Set(cod.HeaderContentType, "application/json")
		c := cod.NewContext(nil, req)
		err := bodyParser(c)
		if err == nil {
			t.Fatalf("read body fail should return error")
		}
	})

	t.Run("body over limit size", func(t *testing.T) {
		bodyParser := NewBodyParser(BodyParserConfig{
			Limit: 1,
		})
		req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader("abc"))
		req.Header.Set(cod.HeaderContentType, "application/json")
		c := cod.NewContext(nil, req)
		err := bodyParser(c)
		if err == nil {
			t.Fatalf("body over size should return error")
		}
	})

	t.Run("ignore json and content type is json", func(t *testing.T) {
		bodyParser := NewBodyParser(BodyParserConfig{
			IgnoreJSON: true,
		})
		req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader("abc"))
		req.Header.Set(cod.HeaderContentType, "application/json")
		c := cod.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := bodyParser(c)
		if err != nil {
			t.Fatalf("body parse fail, %v", err)
		}
		if !done {
			t.Fatalf("body parse fail")
		}
		if len(c.RequestBody) != 0 {
			t.Fatalf("body parse shoudl be pass")
		}
	})

	t.Run("ignore form url encoded and content type is form url encoded", func(t *testing.T) {
		bodyParser := NewBodyParser(BodyParserConfig{
			IgnoreFormURLEncoded: true,
		})
		body := `name=tree.xie&type=1`
		req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader(body))
		req.Header.Set(cod.HeaderContentType, "application/x-www-form-urlencoded")
		c := cod.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := bodyParser(c)
		if err != nil {
			t.Fatalf("form url encoded parse fail, %v", err)
		}
		if !done {
			t.Fatalf("form url encoded parse fail")
		}
		if len(c.RequestBody) != 0 {
			t.Fatalf("body parse shoudl be pass")
		}
	})

	t.Run("parse json success", func(t *testing.T) {
		bodyParser := NewBodyParser(BodyParserConfig{})
		body := `{"name": "tree.xie"}`
		req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader(body))
		req.Header.Set(cod.HeaderContentType, "application/json")
		c := cod.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			if string(c.RequestBody) != body {
				return errors.New("request body is invalid")
			}
			return nil
		}
		err := bodyParser(c)
		if err != nil {
			t.Fatalf("json parse fail, %v", err)
		}
		if !done {
			t.Fatalf("json parse fail")
		}
	})

	t.Run("parse form url encoded success", func(t *testing.T) {
		bodyParser := NewBodyParser(BodyParserConfig{})
		body := `name=tree.xie&type=1`
		req := httptest.NewRequest("POST", "https://aslant.site/", strings.NewReader(body))
		req.Header.Set(cod.HeaderContentType, "application/x-www-form-urlencoded")
		c := cod.NewContext(nil, req)
		done := false
		c.Next = func() error {
			done = true
			if string(c.RequestBody) != `{"name":"tree.xie","type":"1"}` {
				return errors.New("request body is invalid")
			}
			return nil
		}
		err := bodyParser(c)
		if err != nil {
			t.Fatalf("form url encoded parse fail, %v", err)
		}
		if !done {
			t.Fatalf("form url encoded parse fail")
		}
	})
}
