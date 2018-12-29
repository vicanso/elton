package middleware

import (
	"testing"

	"github.com/vicanso/cod"
)

func TestDefaultSkipper(t *testing.T) {
	c := cod.NewContext(nil, nil)
	c.Committed = true
	if !DefaultSkipper(c) {
		t.Fatalf("default skip fail")
	}
}

func TestGzip(t *testing.T) {
	buf := []byte("abcd")
	gzipBuf, err := doGzip(buf, 0)
	if err != nil ||
		len(gzipBuf) == 0 {
		t.Fatalf("do gzip fail, %v", err)
	}
}
