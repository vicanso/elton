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

package elton

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTMLTemplate(t *testing.T) {
	assert := assert.New(t)
	ht := HTMLTemplate{}

	type Data struct {
		ID   int
		Name string
	}

	text := "<p>{{.ID}}<span>{{.Name}}</span></p>"

	t.Run("render text", func(t *testing.T) {
		// render text
		html, err := ht.Render(context.Background(), text, &Data{
			ID:   1,
			Name: "tree.xie",
		})
		assert.Nil(err)
		assert.Equal("<p>1<span>tree.xie</span></p>", html)
	})

	t.Run("render file", func(t *testing.T) {
		// render file
		f, err := os.CreateTemp("", "")
		assert.Nil(err)
		filename := f.Name()
		defer os.Remove(filename)
		_, err = f.WriteString(text)
		assert.Nil(err)
		err = f.Close()
		assert.Nil(err)
		html, err := ht.RenderFile(context.Background(), filename, &Data{
			ID:   2,
			Name: "tree",
		})
		assert.Nil(err)
		assert.Equal("<p>2<span>tree</span></p>", html)
	})
}

func TestTemplates(t *testing.T) {
	assert := assert.New(t)

	assert.NotNil(DefaultTemplateParsers.Get("html"))
	assert.NotNil(DefaultTemplateParsers.Get("tmpl"))
}
