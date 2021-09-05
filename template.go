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
	"io/ioutil"
	"text/template"
)

type TemplateParser interface {
	Render(ctx context.Context, text string, data interface{}) (string, error)
	RenderFile(ctx context.Context, filename string, data interface{}) (string, error)
}
type TemplateParsers map[string]TemplateParser

func (tps TemplateParsers) Add(tmplType string, parser TemplateParser) {
	tps[tmplType] = parser
}
func (tps TemplateParsers) Get(tmplType string) TemplateParser {
	if tps == nil {
		return nil
	}
	return tps[tmplType]
}
func NewTemplateParsers() TemplateParsers {
	return make(TemplateParsers)
}

var _ TemplateParser = (*HTMLTemplate)(nil)
var defaultTemplateParsers = NewTemplateParsers()

func Register(tmplType string, parser TemplateParser) {
	defaultTemplateParsers.Add(tmplType, parser)
}

func init() {
	Register("tmpl", &HTMLTemplate{})
	Register("html", &HTMLTemplate{})
}

func GetParser(tmplType string) TemplateParser {
	return defaultTemplateParsers.Get(tmplType)
}

type HTMLTemplate struct{}

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
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return ht.render(filename, string(buf), data)
}
