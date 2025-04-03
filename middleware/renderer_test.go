// MIT License

// Copyright (c) 2021 Tree Xie

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
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestGetTemplateType(t *testing.T) {
	assert := assert.New(t)
	data := RenderData{}

	// 默认值
	assert.Equal("html", data.getTemplateType())

	// 从文件后缀获取
	data.File = "test.ejs"
	assert.Equal("ejs", data.getTemplateType())

	// 指定类型
	data.TemplateType = "pug"
	assert.Equal("pug", data.getTemplateType())
}

func TestRenderer(t *testing.T) {
	assert := assert.New(t)
	type Data struct {
		ID   int
		Name string
	}

	renderer := NewRenderer(RendererConfig{})
	text := "<p>{{.ID}}<span>{{.Name}}</span></p>"

	t.Run("not set render data", func(t *testing.T) {
		c := elton.NewContext(nil, nil)
		c.Next = func() error {
			return nil
		}
		c.Body = "hello world!"
		err := renderer(c)
		assert.Nil(err)
		assert.Empty(c.BodyBuffer)
	})

	t.Run("file and text is nil", func(t *testing.T) {
		c := elton.NewContext(nil, nil)
		c.Next = func() error {
			return nil
		}
		c.Body = RenderData{}
		err := renderer(c)
		assert.Equal(ErrFileAndTextNil, err)
	})

	t.Run("tempate is not support", func(t *testing.T) {
		c := elton.NewContext(nil, nil)
		c.Next = func() error {
			return nil
		}
		c.Body = RenderData{
			Text:         text,
			TemplateType: "pug",
		}
		err := renderer(c)
		assert.Equal(ErrTemplateTypeInvalid, err)
	})

	t.Run("render html from text", func(t *testing.T) {
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, httptest.NewRequest("GET", "/", nil))
		c.Next = func() error {
			return nil
		}
		c.Body = RenderData{
			Text: text,
			Data: &Data{
				ID:   1,
				Name: "tree.xie",
			},
		}
		err := renderer(c)
		assert.Nil(err)
		assert.Equal("<p>1<span>tree.xie</span></p>", c.BodyBuffer.String())
		assert.Equal("text/html; charset=utf-8", resp.Header().Get(elton.HeaderContentType))
	})

	t.Run("render html from file", func(t *testing.T) {
		// render file
		f, err := os.CreateTemp("", "")
		assert.Nil(err)
		filename := f.Name()
		defer func() {
			_ = os.Remove(filename)
		}()
		_, err = f.WriteString(text)
		assert.Nil(err)
		err = f.Close()
		assert.Nil(err)

		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, httptest.NewRequest("GET", "/", nil))
		c.Next = func() error {
			return nil
		}
		c.Body = &RenderData{
			File: filename,
			Data: &Data{
				ID:   2,
				Name: "tree",
			},
		}
		err = renderer(c)
		assert.Nil(err)
		assert.Equal("<p>2<span>tree</span></p>", c.BodyBuffer.String())
		assert.Equal("text/html; charset=utf-8", resp.Header().Get(elton.HeaderContentType))
	})
}
