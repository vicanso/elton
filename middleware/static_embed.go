// +build go1.16

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
	"embed"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vicanso/elton"
)

type embedStaticFS struct {
	// prefix of file
	Prefix string
	FS     embed.FS
}

// NewEmbedStaticFS resturns a new embed static fs
func NewEmbedStaticFS(fs embed.FS, prefix string) *embedStaticFS {
	return &embedStaticFS{
		Prefix: prefix,
		FS:     fs,
	}
}

func (es *embedStaticFS) getFile(file string) string {
	windowsPathSeparator := "\\"
	return strings.ReplaceAll(filepath.Join(es.Prefix, file), windowsPathSeparator, "/")
}

// Exists check the file exists
func (es *embedStaticFS) Exists(file string) bool {
	f, err := es.FS.Open(es.getFile(file))
	if err != nil {
		return false
	}
	defer f.Close()
	return true
}

// Get returns content of file
func (es *embedStaticFS) Get(file string) ([]byte, error) {
	return es.FS.ReadFile(es.getFile(file))
}

// Stat return nil for file stat
func (es *embedStaticFS) Stat(file string) os.FileInfo {
	// 文件打包至程序中，因此无file info
	return nil
}

// NewReader returns a reader of file
func (es *embedStaticFS) NewReader(file string) (io.Reader, error) {
	buf, err := es.Get(file)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(buf), nil
}

// SendFile sends file to http response and set content type
func (es *embedStaticFS) SendFile(c *elton.Context, file string) (err error) {
	// 因为静态文件打包至程序中，直接读取
	buf, err := es.Get(file)
	if err != nil {
		return
	}
	// 根据文件后续设置类型
	c.SetContentTypeByExt(file)
	c.BodyBuffer = bytes.NewBuffer(buf)
	return
}
