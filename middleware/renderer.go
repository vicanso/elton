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
	"bytes"
	"path/filepath"

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

type RenderData struct {
	File         string
	Text         string
	TemplateType string
	Data         interface{}
}

type RendererConfig struct {
	Skipper elton.Skipper
	Parsers elton.TemplateParsers
}

func (data *RenderData) getTemplateType() string {
	// 获取模板类型
	templateType := data.TemplateType
	if templateType == "" && data.File != "" {
		ext := filepath.Ext(data.File)
		if ext != "" {
			templateType = ext[1:]
		}
	}
	// 默认为html模板
	if templateType == "" {
		templateType = "html"
	}
	return templateType
}

const ErrRendererCategory = "elton-renderer"

var (
	ErrTemplateTypeInvalid = &hes.Error{
		Exception:  true,
		StatusCode: 500,
		Message:    "template type is invalid",
		Category:   ErrRendererCategory,
	}
	ErrFileAndTextNil = &hes.Error{
		Exception:  true,
		StatusCode: 500,
		Message:    "file and text can not be nil",
		Category:   ErrRendererCategory,
	}
)

// NewRenderer returns a new renderer middleware.
// It will render the template with data,
// and set response data as html.
func NewRenderer(config RendererConfig) elton.Handler {
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	parsers := config.Parsers
	if parsers == nil {
		parsers = elton.DefaultTemplateParsers
	}
	return func(c *elton.Context) error {
		err := c.Next()
		if skipper(c) {
			return err
		}
		if err != nil {
			return err
		}
		valid := false
		var data *RenderData
		switch d := c.Body.(type) {
		case *RenderData:
			valid = true
			data = d
		case RenderData:
			valid = true
			data = &d
		default:
			valid = false
		}
		// 如果设置的数据非render data
		// 则直接返回
		if !valid {
			return nil
		}
		// 如果文件和模板均为空
		if data.File == "" && data.Text == "" {
			return ErrFileAndTextNil
		}
		// 获取模板类型
		templateType := data.getTemplateType()

		parser := parsers.Get(templateType)
		if parser == nil {
			return ErrTemplateTypeInvalid
		}
		var html string
		if data.File != "" {
			html, err = parser.RenderFile(c.Context(), data.File, data.Data)
		} else {
			html, err = parser.Render(c.Context(), data.Text, data.Data)
		}
		if err != nil {
			return err
		}
		c.SetContentTypeByExt(".html")
		c.BodyBuffer = bytes.NewBufferString(html)

		return nil
	}
}
