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
	"bytes"
	"context"
	"html/template"
	"io/ioutil"
)

type TemplateParser interface {
	Render(ctx context.Context, text string, data interface{}) (string, error)
	RenderFile(ctx context.Context, filename string, data interface{}) (string, error)
}
type TemplateParsers map[string]TemplateParser

// ReadFile defines how to read file
type ReadFile func(filename string) ([]byte, error)

func (tps TemplateParsers) Add(template string, parser TemplateParser) {
	if template == "" || parser == nil {
		panic("template and parser can not be nil")
	}
	tps[template] = parser
}
func (tps TemplateParsers) Get(template string) TemplateParser {
	if tps == nil || template == "" {
		return nil
	}
	return tps[template]
}
func NewTemplateParsers() TemplateParsers {
	return make(TemplateParsers)
}

var _ TemplateParser = (*HTMLTemplate)(nil)
var DefaultTemplateParsers = NewTemplateParsers()

func init() {
	DefaultTemplateParsers.Add("tmpl", NewHTMLTemplate(nil))
	DefaultTemplateParsers.Add("html", NewHTMLTemplate(nil))
}

func NewHTMLTemplate(read ReadFile) *HTMLTemplate {
	return &HTMLTemplate{
		readFile: read,
	}
}

type HTMLTemplate struct {
	readFile ReadFile
}

func (ht *HTMLTemplate) render(name, text string, data interface{}) (string, error) {
	tpl, err := template.New(name).Parse(text)
	if err != nil {
		return "", err
	}
	b := bytes.Buffer{}
	err = tpl.Execute(&b, data)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

// Render renders the text using text/template
func (ht *HTMLTemplate) Render(ctx context.Context, text string, data interface{}) (string, error) {
	return ht.render("", text, data)
}

// Render renders the text of file using text/template
func (ht *HTMLTemplate) RenderFile(ctx context.Context, filename string, data interface{}) (string, error) {
	read := ht.readFile
	if read == nil {
		read = ioutil.ReadFile
	}
	buf, err := read(filename)
	if err != nil {
		return "", err
	}
	return ht.render(filename, string(buf), data)
}
