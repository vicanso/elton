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
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

type (
	// StaticFile static file
	StaticFile interface {
		Exists(string) bool
		Get(string) ([]byte, error)
		Stat(string) os.FileInfo
		NewReader(string) (io.Reader, error)
	}
	// StaticServeConfig static serve config
	StaticServeConfig struct {
		// 静态文件目录
		Path string
		// http cache control max age
		MaxAge time.Duration
		// http cache control s-maxage
		SMaxAge time.Duration
		// http cache control immutable
		Immutable bool
		// http response header
		Header map[string]string
		// 禁止query string（因为有时静态文件为CDN回源，避免生成各种重复的缓存）
		DenyQueryString bool
		// 是否禁止文件路径以.开头（因为这些文件有可能包括重要信息）
		DenyDot bool
		// 是否使用strong eTag
		EnableStrongETag bool
		// 禁止生成ETag
		DisableETag bool
		// 禁止生成 last-modifed
		DisableLastModified bool
		// 如果404，是否调用next执行后续的中间件（默认为不执行，返回404错误）
		NotFoundNext bool
		// 符合该正则则设置为no cache
		NoCacheRegexp *regexp.Regexp
		// 响应前的处理(只针对读取到buffer的文件)
		BeforeResponse func(string, []byte) ([]byte, error)
		// 目录默认文件
		IndexFile string
		Skipper   elton.Skipper
	}
	// FS file system
	FS struct {
	}
)

const (
	// ErrStaticServeCategory static serve error category
	ErrStaticServeCategory = "elton-static-serve"
)

var (
	// ErrStaticServeNotAllowQueryString not all query string
	ErrStaticServeNotAllowQueryString = getStaticServeError("static serve not allow query string", http.StatusBadRequest)
	// ErrStaticServeNotFound static file not found
	ErrStaticServeNotFound = getStaticServeError("static file not found", http.StatusNotFound)
	// ErrStaticServeOutOfPath file out of path
	ErrStaticServeOutOfPath = getStaticServeError("out of path", http.StatusBadRequest)
	// ErrStaticServeNotAllowAccessDot file include dot
	ErrStaticServeNotAllowAccessDot = getStaticServeError("static server not allow with dot", http.StatusBadRequest)
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
func (fs *FS) Get(file string) ([]byte, error) {
	return os.ReadFile(file)
}

// NewReader new a reader for file
func (fs *FS) NewReader(file string) (io.Reader, error) {
	return os.Open(file)
}

// getStaticServeError 获取static serve的出错
func getStaticServeError(message string, statusCode int) *hes.Error {
	return &hes.Error{
		StatusCode: statusCode,
		Message:    message,
		Category:   ErrStaticServeCategory,
	}
}

// generateETag generate eTag
func generateETag(buf []byte) string {
	size := len(buf)
	if size == 0 {
		return `"0-2jmj7l5rSw0yVb_vlWAYkK_YBwk="`
	}
	h := sha1.New()
	_, err := h.Write(buf)
	if err != nil {
		return ""
	}
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf(`"%x-%s"`, size, hash)
}

// NewDefaultStaticServe returns a new default static server milldeware using FS
func NewDefaultStaticServe(config StaticServeConfig) elton.Handler {
	return NewStaticServe(&FS{}, config)
}

func toSeconds(d time.Duration) string {
	return strconv.Itoa(int(d.Seconds()))
}

// NewStaticServe returns a new static serve middleware, suggest to set the MaxAge and SMaxAge for cache control for better performance.
// It will return an error if DenyDot is true and filename is start with '.'.
// It will return an error if DenyQueryString is true and the querystring is not empty.
func NewStaticServe(staticFile StaticFile, config StaticServeConfig) elton.Handler {
	cacheArr := []string{
		"public",
	}
	if config.MaxAge > 0 {
		cacheArr = append(cacheArr, "max-age="+toSeconds(config.MaxAge))
	}
	if config.SMaxAge > 0 {
		cacheArr = append(cacheArr, "s-maxage="+toSeconds(config.SMaxAge))
	}
	cacheControl := ""
	if len(cacheArr) > 1 {
		cacheControl = strings.Join(cacheArr, ", ")
	}
	if cacheControl != "" && config.Immutable {
		cacheControl += ", immutable"
	}
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	// convert different os file path
	basePath := filepath.Join(config.Path, "")
	noCacheRegexp := config.NoCacheRegexp
	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		file := ""
		// 从第一个参数获取文件名
		if c.Params != nil && len(c.Params.Values) > 0 {
			file = c.Params.Values[0]
		}

		url := c.Request.URL

		if file == "" {
			file = url.Path
		}

		file = filepath.Join(config.Path, file)
		// 避免文件名是有 .. 等导致最终文件路径越过配置的路径
		if !strings.HasPrefix(file, basePath) {
			return ErrStaticServeOutOfPath
		}

		// 检查文件（路径）是否包括.
		if config.DenyDot {
			arr := strings.SplitN(file, string(filepath.Separator), -1)
			for _, item := range arr {
				if item != "" && item[0] == '.' {
					return ErrStaticServeNotAllowAccessDot
				}
			}
		}

		// 禁止 querystring
		if config.DenyQueryString && url.RawQuery != "" {
			return ErrStaticServeNotAllowQueryString
		}
		// 如果有配置目录的index文件
		if config.IndexFile != "" {
			fileInfo := staticFile.Stat(file)
			if fileInfo != nil && fileInfo.IsDir() {
				file = filepath.Join(file, config.IndexFile)
			}
		}

		exists := staticFile.Exists(file)
		if !exists {
			if config.NotFoundNext {
				return c.Next()
			}
			return ErrStaticServeNotFound
		}

		c.SetContentTypeByExt(file)
		var fileBuf []byte
		// strong eTag需要读取文件内容计算eTag
		if !config.DisableETag && config.EnableStrongETag {
			buf, e := staticFile.Get(file)
			if e != nil {
				he, ok := e.(*hes.Error)
				if !ok {
					he = hes.NewWithErrorStatusCode(e, http.StatusInternalServerError)
					he.Category = ErrStaticServeCategory
				}
				return he
			}
			fileBuf = buf
		}

		if !config.DisableETag {
			if config.EnableStrongETag {
				eTag := generateETag(fileBuf)
				if eTag != "" {
					c.SetHeader(elton.HeaderETag, eTag)
				}
			} else {
				fileInfo := staticFile.Stat(file)
				if fileInfo != nil {
					eTag := fmt.Sprintf(`W/"%x-%x"`, fileInfo.Size(), fileInfo.ModTime().Unix())
					c.SetHeader(elton.HeaderETag, eTag)
				}
			}
		}

		if !config.DisableLastModified {
			fileInfo := staticFile.Stat(file)
			if fileInfo != nil {
				lmd := fileInfo.ModTime().UTC().Format(time.RFC1123)
				c.SetHeader(elton.HeaderLastModified, lmd)
			}
		}

		for k, v := range config.Header {
			c.AddHeader(k, v)
		}
		// 如果有设置before response
		if config.BeforeResponse != nil && fileBuf != nil {
			buf, err := config.BeforeResponse(file, fileBuf)
			if err != nil {
				return err
			}
			fileBuf = buf
		}
		// 未设置cache control
		// 或文件符合正则
		if cacheControl == "" ||
			(noCacheRegexp != nil && noCacheRegexp.MatchString(file)) {
			c.NoCache()
		} else {
			c.SetHeader(elton.HeaderCacheControl, cacheControl)
		}
		if fileBuf != nil {
			c.StatusCode = http.StatusOK
			c.BodyBuffer = bytes.NewBuffer(fileBuf)
		} else {
			r, e := staticFile.NewReader(file)
			if e != nil {
				return getStaticServeError(e.Error(), http.StatusBadRequest)
			}
			c.StatusCode = http.StatusOK
			c.Body = r
		}
		return c.Next()
	}
}
