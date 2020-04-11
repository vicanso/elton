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
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

var testData []byte

func init() {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	fn := func(n int) string {
		b := make([]rune, n)
		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}
		return string(b)
	}
	testData = []byte(fn(4096))
}

func TestGen(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(genETag([]byte("")), `"0-2jmj7l5rSw0yVb_vlWAYkK_YBwk="`)
}

func TestETag(t *testing.T) {
	fn := NewDefaultETag()
	t.Run("curstom error", func(t *testing.T) {
		assert := assert.New(t)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, nil)
		customErr := errors.New("abcd")
		c.Next = func() error {
			return customErr
		}
		err := fn(c)
		assert.Equal(err, customErr)
	})

	t.Run("no body", func(t *testing.T) {
		assert := assert.New(t)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, nil)
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.Empty(c.GetHeader(elton.HeaderETag))
	})

	t.Run("error status", func(t *testing.T) {
		assert := assert.New(t)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, nil)
		c.Next = func() error {
			c.Body = map[string]string{
				"name": "tree.xie",
			}
			c.StatusCode = 400
			c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.Empty(c.GetHeader(elton.HeaderETag))
	})

	t.Run("gen eTag", func(t *testing.T) {
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, nil)
		c.Next = func() error {
			c.Body = map[string]string{
				"name": "tree.xie",
			}
			c.BodyBuffer = bytes.NewBufferString(`{"name":"tree.xie"}`)
			return nil
		}
		err := fn(c)
		if err != nil {
			t.Fatalf("eTag middleware fail, %v", err)
		}
		if c.GetHeader(elton.HeaderETag) != `"13-yo9YroUOjW1obRvVoXfrCiL2JGE="` {
			t.Fatalf("gen eTag fail")
		}
	})
}

func BenchmarkGenETag(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		genETag(testData)
	}
}

func BenchmarkMd5(b *testing.B) {
	b.ReportAllocs()
	fn := func(buf []byte) string {
		size := len(buf)
		if size == 0 {
			return `"0-2jmj7l5rSw0yVb_vlWAYkK_YBwk="`
		}
		h := md5.New()
		_, _ = h.Write(buf)
		hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
		return fmt.Sprintf(`"%x-%s"`, size, hash)
	}
	for i := 0; i < b.N; i++ {
		fn(testData)
	}
}
