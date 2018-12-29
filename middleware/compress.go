package middleware

import (
	"regexp"
	"strings"

	"github.com/vicanso/cod"
)

var (
	defaultCompressRegexp = regexp.MustCompile("text|javascript|json")
)

const (
	defaultCompresMinLength = 1024
	gzipCompress            = "gzip"
)

type (
	// CustomCompress custom compress function
	CustomCompress func(*cod.Context) bool
	// CompressConfig compress config
	CompressConfig struct {
		// Level 压缩率级别
		Level int
		// MinLength 最小压缩长度
		MinLength int
		// Checker 校验数据是否可压缩
		Checker   *regexp.Regexp
		Skipper   Skipper
		Compresss CustomCompress
	}
)

// NewCompresss create a new compress middleware
func NewCompresss(config CompressConfig) cod.Handler {
	minLength := config.MinLength
	if minLength == 0 {
		minLength = defaultCompresMinLength
	}
	skiper := config.Skipper
	if skiper == nil {
		skiper = DefaultSkipper
	}
	checker := config.Checker
	if checker == nil {
		checker = defaultCompressRegexp
	}
	customCompress := config.Compresss
	return func(c *cod.Context) (err error) {
		if skiper(c) {
			return c.Next()
		}
		err = c.Next()
		if err != nil {
			return
		}
		respHeader := c.Headers
		encoding := respHeader.Get(cod.HeaderContentEncoding)
		// encoding 不为空，已做处理，无需要压缩
		if encoding != "" {
			return
		}
		contentType := respHeader.Get(cod.HeaderContentType)
		buf := c.BodyBytes
		// 如果数据长度少于最小压缩长度或数据类型为非可压缩，则返回
		if len(buf) < minLength || !checker.MatchString(contentType) {
			return
		}

		// 如果有自定义压缩函数，并处理结果为已完成
		if customCompress != nil && customCompress(c) {
			return
		}

		acceptEncoding := c.GetRequestHeader(cod.HeaderAcceptEncoding)
		// 如果请求端不接受gzip，则返回
		if !strings.Contains(acceptEncoding, gzipCompress) {
			return
		}
		gzipBuf, e := doGzip(buf, config.Level)
		// 如果压缩成功，则使用压缩数据
		// 失败则忽略
		if e == nil {
			c.SetHeader(cod.HeaderContentEncoding, gzipCompress)
			c.BodyBytes = gzipBuf
		}

		return
	}
}
