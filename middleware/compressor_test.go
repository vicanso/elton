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
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestCompressor(t *testing.T) {
	tests := []struct {
		compressor Compressor
		encoding   string
		uncompress func([]byte) ([]byte, error)
	}{
		{
			compressor: new(GzipCompressor),
			encoding:   GzipEncoding,
			uncompress: func(b []byte) ([]byte, error) {
				r, err := gzip.NewReader(bytes.NewReader(b))
				if err != nil {
					return nil, err
				}
				defer r.Close()
				return ioutil.ReadAll(r)
			},
		},
		{
			compressor: new(BrCompressor),
			encoding:   BrEncoding,
			uncompress: func(b []byte) ([]byte, error) {
				r := brotli.NewReader(bytes.NewReader(b))
				return ioutil.ReadAll(r)
			},
		},
	}
	assert := assert.New(t)
	for _, tt := range tests {
		originalData := randomString(1024)
		req := httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		c := elton.NewContext(nil, req)

		acceptable, encoding := tt.compressor.Accept(c, 0)
		assert.False(acceptable)
		assert.Empty(encoding)

		acceptable, encoding = tt.compressor.Accept(c, len(originalData))
		assert.True(acceptable)
		assert.Equal(tt.encoding, encoding)

		_, err := tt.compressor.Compress([]byte(originalData), 1)
		assert.Nil(err)
		buffer, err := tt.compressor.Compress([]byte(originalData), IgnoreCompression)
		assert.Nil(err)
		assert.Nil(err)

		uncompressBuf, _ := tt.uncompress(buffer.Bytes())
		assert.Equal([]byte(originalData), uncompressBuf)
	}
}

func TestGzipPipe(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		compressor Compressor
		encoding   string
		uncompress func(io.Reader) ([]byte, error)
	}{
		{
			compressor: new(GzipCompressor),
			uncompress: func(r io.Reader) ([]byte, error) {
				gzipReader, err := gzip.NewReader(r)
				if err != nil {
					return nil, err
				}
				defer gzipReader.Close()
				return ioutil.ReadAll(gzipReader)
			},
		},
		{
			compressor: new(BrCompressor),
			uncompress: func(r io.Reader) ([]byte, error) {
				return ioutil.ReadAll(brotli.NewReader(r))
			},
		},
	}
	for _, tt := range tests {
		resp := httptest.NewRecorder()
		originalData := randomString(1024)
		c := elton.NewContext(resp, nil)

		c.Body = bytes.NewReader([]byte(originalData))

		err := tt.compressor.Pipe(c)
		assert.Nil(err)
		buf, _ := tt.uncompress(resp.Body)
		assert.Equal([]byte(originalData), buf)
	}

}

func TestCompressorGetLevel(t *testing.T) {
	assert := assert.New(t)
	g := GzipCompressor{
		Level: 1000,
	}
	assert.Equal(gzip.BestCompression, g.getLevel())
	g.Level = 0
	assert.Equal(gzip.DefaultCompression, g.getLevel())
	g.Level = 1
	assert.Equal(1, g.getLevel())

	br := BrCompressor{
		Level: 1000,
	}
	assert.Equal(maxBrQuality, br.getLevel())
	br.Level = 0
	assert.Equal(defaultBrQuality, br.getLevel())
	br.Level = 1
	assert.Equal(1, br.getLevel())
}

func TestCompressorGetMinLength(t *testing.T) {
	assert := assert.New(t)

	g := GzipCompressor{}
	assert.Equal(DefaultCompressMinLength, g.getMinLength())
	g.MinLength = 1
	assert.Equal(1, g.getMinLength())

	br := BrCompressor{}
	assert.Equal(DefaultCompressMinLength, br.getMinLength())
	br.MinLength = 1
	assert.Equal(1, br.getMinLength())
}
