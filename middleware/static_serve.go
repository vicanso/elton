// Copyright 2018 tree xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package middleware

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vicanso/cod"
	"github.com/vicanso/hes"
)

type (
	// StaticFile static file
	StaticFile interface {
		Exists(string) bool
		Get(string) ([]byte, error)
		Stat(string) os.FileInfo
	}
	// StaticServeConfig static serve config
	StaticServeConfig struct {
		// 静态文件目录
		Path string
		// mount path
		Mount string
		// http cache control max age
		MaxAge int
		// http cache control s-maxage
		SMaxAge int
		// http response header
		Header map[string]string
		// 禁止query string（因为有时静态文件为CDN回源，避免生成各种重复的缓存）
		DenyQueryString bool
		// 是否禁止文件路径以.开头（因为这些文件有可能包括重要信息）
		DenyDot bool
		// 禁止生成ETag
		DisableETag bool
		// 禁止生成 last-modifed
		DisableLastModified bool
		// 如果404，是否调用next执行后续的中间件（默认为不执行，返回404错误）
		NotFoundNext bool
		Skipper      Skipper
	}
	// FS file system
	FS struct {
	}
)

const (
	errStaticServeCategory = "cod-static-serve"
)

var (
	// ErrNotAllowQueryString not all query string
	ErrNotAllowQueryString = getStaticServeError("static serve not allow query string", http.StatusBadRequest)
	// ErrNotFound static file not found
	ErrNotFound = getStaticServeError("static file not found", http.StatusNotFound)
	// ErrOutOfPath file out of path
	ErrOutOfPath = getStaticServeError("out of path", http.StatusBadRequest)
	// ErrNotAllowAccessDot file include dot
	ErrNotAllowAccessDot = getStaticServeError("static server not allow with dot", http.StatusBadRequest)
)

// Exists check the file exists
func (fs *FS) Exists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

// Stat get stat of file
func (fs *FS) Stat(file string) os.FileInfo {
	info, _ := os.Stat(file)
	return info
}

// Get get the file's content
func (fs *FS) Get(file string) (buf []byte, err error) {
	buf, err = ioutil.ReadFile(file)
	return
}

// getStaticServeError 获取static serve的出错
func getStaticServeError(message string, statusCode int) *hes.Error {
	return &hes.Error{
		StatusCode: statusCode,
		Message:    message,
		Category:   errStaticServeCategory,
	}
}

// NewStaticServe create a static serve middleware
func NewStaticServe(staticFile StaticFile, config StaticServeConfig) cod.Handler {
	if config.Path == "" {
		panic("require static path")
	}
	mount := config.Mount
	mountLength := len(mount)
	cacheArr := []string{
		"public",
	}
	if config.MaxAge > 0 {
		cacheArr = append(cacheArr, "max-age="+strconv.Itoa(config.MaxAge))
	}
	if config.SMaxAge > 0 {
		cacheArr = append(cacheArr, "s-maxage="+strconv.Itoa(config.SMaxAge))
	}
	cacheControl := ""
	if len(cacheArr) > 1 {
		cacheControl = strings.Join(cacheArr, ", ")
	}
	skipper := config.Skipper
	if skipper == nil {
		skipper = DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		url := c.Request.URL

		file := url.Path
		if mount != "" {
			// 如果指定了mount，但请求不是以mount开头，则跳过
			if !strings.HasPrefix(file, mount) {
				return c.Next()
			}
			file = file[mountLength:]
		}

		// 检查文件（路径）是否包括.
		if config.DenyDot {
			arr := strings.SplitN(file, "/", -1)
			for _, item := range arr {
				if item != "" && item[0] == '.' {
					err = ErrNotAllowAccessDot
					return
				}
			}
		}

		file = filepath.Join(config.Path, file)
		if !strings.HasPrefix(file, config.Path) {
			err = ErrOutOfPath
			return
		}

		if config.DenyQueryString && url.RawQuery != "" {
			err = ErrNotAllowQueryString
			return
		}
		exists := staticFile.Exists(file)
		if !exists {
			if config.NotFoundNext {
				return c.Next()
			}
			err = ErrNotFound
			return
		}

		c.SetFileContentType(file)
		buf, e := staticFile.Get(file)
		if e != nil {
			he, ok := e.(*hes.Error)
			if ok {
				err = he
			} else {
				err = getStaticServeError(e.Error(), http.StatusInternalServerError)
			}
			return
		}
		if !config.DisableETag {
			eTag := cod.GenerateETag(buf)
			c.SetHeader(cod.HeaderETag, eTag)
		}
		if !config.DisableLastModified {
			fileInfo := staticFile.Stat(file)
			if fileInfo != nil {
				lmd := fileInfo.ModTime().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
				c.SetHeader(cod.HeaderLastModified, lmd)
			}
		}

		for k, v := range config.Header {
			c.SetHeader(k, v)
		}
		if cacheControl != "" {
			c.SetHeader(cod.HeaderCacheControl, cacheControl)
		}
		c.BodyBuffer = bytes.NewBuffer(buf)
		return c.Next()
	}
}
