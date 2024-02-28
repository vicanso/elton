//go:build go1.16
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
	"archive/tar"
	"bytes"
	"embed"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

type embedStaticFS struct {
	// prefix of file
	Prefix string
	FS     embed.FS
}

var _ StaticFile = (*embedStaticFS)(nil)

// NewEmbedStaticFS returns a new embed static fs
func NewEmbedStaticFS(fs embed.FS, prefix string) *embedStaticFS {
	return &embedStaticFS{
		Prefix: prefix,
		FS:     fs,
	}
}

func getFile(prefix string, file string) string {
	windowsPathSeparator := "\\"
	return strings.ReplaceAll(filepath.Join(prefix, file), windowsPathSeparator, "/")
}

func (es *embedStaticFS) getFile(file string) string {
	return getFile(es.Prefix, file)
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
func (es *embedStaticFS) SendFile(c *elton.Context, file string) error {
	// 因为静态文件打包至程序中，直接读取
	buf, err := es.Get(file)
	if err != nil {
		return err
	}
	// 根据文件后续设置类型
	c.SetContentTypeByExt(file)
	c.BodyBuffer = bytes.NewBuffer(buf)
	return nil
}

type tarFS struct {
	// prefix of file
	Prefix string
	// tar file
	File string
	// embed fs
	Embed *embed.FS
}

var _ StaticFile = (*tarFS)(nil)

// NewTarFS returns a new tar static fs
func NewTarFS(file string) *tarFS {
	return &tarFS{
		File: file,
	}
}

func (t *tarFS) get(file string, includeContent bool) (bool, []byte, error) {
	var f fs.File
	var err error
	if t.Embed != nil {
		f, err = t.Embed.Open(t.File)
	} else {
		f, err = os.Open(t.File)
	}
	if err != nil {
		return false, nil, err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	var data []byte
	found := false
	file = getFile(t.Prefix, file)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, nil, err
		}
		if hdr.Name == file {
			found = true
			if includeContent {
				buf, err := io.ReadAll(tr)
				if err != nil {
					return false, nil, err
				}
				data = buf
			}
			break
		}
	}
	return found, data, nil
}

// Exists check the file exists
func (t *tarFS) Exists(file string) bool {
	found, _, _ := t.get(file, false)
	return found
}

// Get returns content of file
func (t *tarFS) Get(file string) ([]byte, error) {
	found, data, err := t.get(file, true)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, hes.NewWithStatusCode("Not Found", 404)
	}
	return data, nil
}

// Stat return nil for file stat
func (t *tarFS) Stat(file string) os.FileInfo {
	return nil
}

// NewReader returns a reader of file
func (t *tarFS) NewReader(file string) (io.Reader, error) {
	buf, err := t.Get(file)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(buf), nil
}
