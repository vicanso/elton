package middleware

import (
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vicanso/cod"
	"github.com/vicanso/errors"
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
		Path                string
		Mount               string
		MaxAge              int
		SMaxAge             int
		Header              map[string]string
		DenyQueryString     bool
		DisableETag         bool
		DisableLastModified bool
		NotFoundNext        bool
		Skipper             Skipper
	}
	// FS file system
	FS struct {
		Path string
	}
)

const (
	errStaticServeCategory = "cod-static-serve"
)

var (
	errNotAllowQueryString = getStaticServeError("static serve not allow query string", http.StatusBadRequest)
	errNotFound            = getStaticServeError("static file not found", http.StatusNotFound)
	errOutOfPath           = getStaticServeError("out of path", http.StatusBadRequest)
)

func (fs *FS) outOfPath(file string) bool {
	// 判断是否超时指定目录
	if fs.Path == "" || strings.HasPrefix(file, fs.Path) {
		return false
	}
	return true
}

// Exists check the file exists
func (fs *FS) Exists(file string) bool {
	// 如果非指定目录下的文件，则认为不存在
	if fs.outOfPath(file) {
		return false
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

// Stat get stat of file
func (fs *FS) Stat(file string) os.FileInfo {
	if fs.outOfPath(file) {
		return nil
	}
	info, _ := os.Stat(file)
	return info
}

// Get get the file's content
func (fs *FS) Get(file string) (buf []byte, err error) {
	if fs.outOfPath(file) {
		return nil, errOutOfPath
	}
	buf, err = ioutil.ReadFile(file)
	return
}

// getStaticServeError 获取static serve的出错
func getStaticServeError(message string, statusCode int) *errors.HTTPError {
	return &errors.HTTPError{
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
	skiper := config.Skipper
	if skiper == nil {
		skiper = DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skiper(c) {
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
		file = filepath.Join(config.Path, file)

		if config.DenyQueryString && url.RawQuery != "" {
			err = errNotAllowQueryString
			return
		}
		exists := staticFile.Exists(file)
		if !exists {
			if config.NotFoundNext {
				return c.Next()
			}
			err = errNotFound
			return
		}

		contentType := mime.TypeByExtension(filepath.Ext(file))
		if contentType != "" {
			c.SetHeader(cod.HeaderContentType, contentType)
		}
		buf, e := staticFile.Get(file)
		if e != nil {
			err = getStaticServeError(e.Error(), http.StatusInternalServerError)
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
		c.Body = buf
		return c.Next()
	}
}
