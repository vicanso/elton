package middleware

import (
	"bytes"
	"compress/gzip"

	"github.com/vicanso/cod"
)

type (
	// Skipper check for skip middleware
	Skipper func(c *cod.Context) bool
)

// DefaultSkipper default skiper function(not skip)
func DefaultSkipper(c *cod.Context) bool {
	return c.Committed
}

// doGzip 对数据压缩
func doGzip(buf []byte, level int) ([]byte, error) {
	var b bytes.Buffer
	if level <= 0 {
		level = gzip.DefaultCompression
	}
	w, _ := gzip.NewWriterLevel(&b, level)
	_, err := w.Write(buf)
	if err != nil {
		return nil, err
	}
	w.Close()
	return b.Bytes(), nil
}
