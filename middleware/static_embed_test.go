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
	"embed"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton/v2"
)

//go:embed *
var assetFS embed.FS

func TestEmbedGetFile(t *testing.T) {
	assert := assert.New(t)
	es := EmbedStaticFS{
		Prefix: "web",
	}
	file := es.getFile("abc\\test.txt")
	assert.Equal("web/abc/test.txt", file)
	file = es.getFile("abc/test.txt")
	assert.Equal("web/abc/test.txt", file)

}
func TestEmbedStaticFS(t *testing.T) {
	assert := assert.New(t)
	file := "static_embed.go"

	fs := NewEmbedStaticFS(assetFS, "")

	assert.True(fs.Exists(file))

	fileInfo := fs.Stat(file)
	assert.Nil(fileInfo)

	buf, err := fs.Get(file)
	assert.Nil(err)
	assert.NotEmpty(buf)

	r, err := fs.NewReader(file)
	assert.Nil(err)
	assert.NotEmpty(r)

	// SendFile
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	c := elton.NewContext(resp, req)
	err = fs.SendFile(c, file)
	assert.Nil(err)
	assert.NotNil(c.BodyBuffer)
	assert.NotEmpty(c.BodyBuffer.Bytes())
}

func TestNewEmbedStaticServe(t *testing.T) {
	assert := assert.New(t)
	h := NewEmbedStaticServe(assetFS, StaticServeConfig{
		// 路由参数会拼到 Path 上；用空 Path 时直接以 URL/param 为 embed 内路径
		EnableStrongETag: true,
	})
	assert.NotNil(h)

	// 覆盖 static_serve.go 内 EmbedFS 实现（与 EmbedStaticFS 不同）
	efs := &EmbedFS{fs: assetFS}
	file := "static_embed.go"
	assert.True(efs.Exists(file))
	assert.False(efs.Exists("no-such-file-xyz"))
	assert.Nil(efs.Stat(file))
	buf, err := efs.Get(file)
	assert.Nil(err)
	assert.NotEmpty(buf)
	r, err := efs.NewReader(file)
	assert.Nil(err)
	assert.NotNil(r)
	_ = r.Close()
}

func TestTarFS(t *testing.T) {
	assert := assert.New(t)

	// 构造临时 tar：含 prefix/hello.txt
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "assets.tar")
	f, err := os.Create(tarPath)
	assert.Nil(err)
	tw := tar.NewWriter(f)
	content := []byte("hello from tar")
	hdr := &tar.Header{
		Name: "web/hello.txt",
		Mode: 0o644,
		Size: int64(len(content)),
	}
	assert.Nil(tw.WriteHeader(hdr))
	_, err = tw.Write(content)
	assert.Nil(err)
	assert.Nil(tw.Close())
	assert.Nil(f.Close())

	tfs := NewTarFS(tarPath)
	tfs.Prefix = "web"

	assert.True(tfs.Exists("hello.txt"))
	assert.False(tfs.Exists("missing.txt"))
	assert.Nil(tfs.Stat("hello.txt"))

	buf, err := tfs.Get("hello.txt")
	assert.Nil(err)
	assert.Equal(content, buf)

	r, err := tfs.NewReader("hello.txt")
	assert.Nil(err)
	got, err := io.ReadAll(r)
	assert.Nil(err)
	assert.Equal(content, got)
	_ = r.Close()

	_, err = tfs.Get("nope.txt")
	assert.NotNil(err)

	// 不存在的 tar 文件
	missing := NewTarFS(filepath.Join(dir, "no-such.tar"))
	assert.False(missing.Exists("hello.txt"))
}
