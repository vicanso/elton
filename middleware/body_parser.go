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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/hes"
)

const (
	// ErrBodyParserCategory body parser error category
	ErrBodyParserCategory = "elton-body-parser"
	// 默认为50kb
	defaultRequestBodyLimit   = 50 * 1024
	jsonContentType           = "application/json"
	formURLEncodedContentType = "application/x-www-form-urlencoded"
)

type (
	// BodyContentTypeValidate body content type check validate function
	BodyContentTypeValidate func(c *elton.Context) bool
	// BodyDecoder body decoder
	BodyDecoder interface {
		// body decode function
		Decode(c *elton.Context, originalData []byte) (data []byte, err error)
		// validate function
		Validate(c *elton.Context) bool
	}
	// BodyParserConfig body parser config
	BodyParserConfig struct {
		// Limit the limit size of body
		Limit int
		// InitCap the initial capacity of buffer
		InitCap int
		// Decoders decode list
		Decoders            []BodyDecoder
		Skipper             elton.Skipper
		ContentTypeValidate BodyContentTypeValidate
		// OnBeforeDecode before decode event
		OnBeforeDecode func(*elton.Context) error
	}

	// gzip decoder
	gzipDecoder struct{}
	// json decoder
	jsonDecoder struct{}
	// form url encoded decoder
	formURLEncodedDecoder struct{}
)

var (
	validMethods = []string{
		http.MethodPost,
		http.MethodPatch,
		http.MethodPut,
	}
	ErrInvalidJSON = &hes.Error{
		Category:   ErrBodyParserCategory,
		Message:    "invalid json format",
		StatusCode: http.StatusBadRequest,
	}
	jsonBytes = []byte("{}[]")
)

func (gd *gzipDecoder) Validate(c *elton.Context) bool {
	return c.GetRequestHeader(elton.HeaderContentEncoding) == elton.Gzip
}

func (gd *gzipDecoder) Decode(c *elton.Context, originalData []byte) (data []byte, err error) {
	c.SetRequestHeader(elton.HeaderContentEncoding, "")
	c.SetRequestHeader(elton.HeaderContentLength, "")
	return doGunzip(originalData)
}

func (jd *jsonDecoder) Validate(c *elton.Context) bool {
	mimeType, _, _ := strings.Cut(c.GetRequestHeader(elton.HeaderContentType), ";")
	return mimeType == jsonContentType
}
func (jd *jsonDecoder) Decode(c *elton.Context, originalData []byte) ([]byte, error) {
	originalData = bytes.TrimSpace(originalData)
	if len(originalData) == 0 {
		return nil, nil
	}
	firstByte := originalData[0]
	lastByte := originalData[len(originalData)-1]

	if firstByte != jsonBytes[0] && firstByte != jsonBytes[2] {
		return nil, ErrInvalidJSON
	}
	if firstByte == jsonBytes[0] && lastByte != jsonBytes[1] {
		return nil, ErrInvalidJSON
	}
	if firstByte == jsonBytes[2] && lastByte != jsonBytes[3] {
		return nil, ErrInvalidJSON
	}
	return originalData, nil
}

func (fd *formURLEncodedDecoder) Validate(c *elton.Context) bool {
	mimeType, _, _ := strings.Cut(c.GetRequestHeader(elton.HeaderContentType), ";")
	return mimeType == formURLEncodedContentType
}

func (fd *formURLEncodedDecoder) Decode(c *elton.Context, originalData []byte) ([]byte, error) {
	urlValues, err := url.ParseQuery(string(originalData))
	if err != nil {
		return nil, hes.Wrap(err)
	}

	// 经 json.Marshal 序列化，避免手工拼接导致的转义缺失与字段注入
	m := make(map[string]any, len(urlValues))
	for key, values := range urlValues {
		if len(values) < 2 {
			if len(values) == 0 {
				m[key] = ""
			} else {
				m[key] = values[0]
			}
			continue
		}
		// 同 key 多值保持为字符串数组；单值与多值类型仍可能不一致（form 固有问题）
		m[key] = values
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil, hes.Wrap(err)
	}
	return data, nil
}

// AddDecoder to body parser config
func (conf *BodyParserConfig) AddDecoder(decoder BodyDecoder) {
	conf.Decoders = append(conf.Decoders, decoder)
}

// NewGzipDecoder returns a new gzip decoder
func NewGzipDecoder() BodyDecoder {
	return &gzipDecoder{}
}

// NewJSONDecoder returns a new json decoder, it only support application/json
func NewJSONDecoder() BodyDecoder {
	return &jsonDecoder{}
}

// NewFormURLEncodedDecoder returns a new url encoded decoder, it only support application/x-www-form-urlencoded
func NewFormURLEncodedDecoder() BodyDecoder {
	return &formURLEncodedDecoder{}
}

// DefaultJSONContentTypeValidate for validate json content type
func DefaultJSONContentTypeValidate(c *elton.Context) bool {
	ct := c.GetRequestHeader(elton.HeaderContentType)
	return strings.HasPrefix(ct, jsonContentType)
}

// DefaultJSONAndFormContentTypeValidate for validate json content type and form url encoded content type
func DefaultJSONAndFormContentTypeValidate(c *elton.Context) bool {
	ct := c.GetRequestHeader(elton.HeaderContentType)
	if strings.HasPrefix(ct, jsonContentType) {
		return true
	}
	return strings.HasPrefix(ct, formURLEncodedContentType)
}

// NewDefaultBodyParser returns a new default body parser, which include gzip and json decoder.
// The body size is limited to 50KB.
func NewDefaultBodyParser() elton.Handler {
	conf := BodyParserConfig{
		ContentTypeValidate: DefaultJSONContentTypeValidate,
	}
	// 如果是压缩的，先解压
	conf.AddDecoder(NewGzipDecoder())
	conf.AddDecoder(NewJSONDecoder())
	return NewBodyParser(conf)
}

// doGunzip gunzip
func doGunzip(buf []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = r.Close()
	}()
	return io.ReadAll(r)
}

type maxBytesReader struct {
	r   io.ReadCloser // underlying reader
	max int64
	n   int64 // max bytes remaining
	err error // sticky error
}

func (l *maxBytesReader) Read(p []byte) (n int, err error) {
	if l.err != nil {
		return 0, l.err
	}
	if len(p) == 0 {
		return 0, nil
	}
	// If they asked for a 32KB byte read but only 5 bytes are
	// remaining, no need to read 32KB. 6 bytes will answer the
	// question of the whether we hit the limit or go past it.
	if int64(len(p)) > l.n+1 {
		p = p[:l.n+1]
	}
	n, err = l.r.Read(p)

	if int64(n) <= l.n {
		l.n -= int64(n)
		l.err = err
		return n, err
	}

	l.err = fmt.Errorf("request body is too large, it should be <= %d", l.max)

	n = int(l.n)
	l.n = 0

	return n, l.err
}

func (l *maxBytesReader) Close() error {
	return l.r.Close()
}

// MaxBytesReader returns a reader which limits the max size of the data,
// it works like http.MaxBytesReader
func MaxBytesReader(r io.ReadCloser, n int64) io.ReadCloser {
	return &maxBytesReader{
		max: n,
		n:   n,
		r:   r,
	}
}

type requestBodyReader struct {
	bufferPool elton.BufferPool
	limit      int
}

func (rr *requestBodyReader) ReadAll(c *elton.Context) ([]byte, error) {
	r := c.Request.Body
	limit := rr.limit
	if limit > 0 {
		r = MaxBytesReader(r, int64(limit))
	}
	defer func() {
		_ = r.Close()
	}()
	var body []byte
	var err error
	contentLength := c.GetRequestHeader(elton.HeaderContentLength)
	// 有合法 Content-Length 时预分配；容量不得超过 limit，避免恶意 CL 放大内存
	initCap := 0
	if contentLength != "" {
		if n, parseErr := strconv.Atoi(contentLength); parseErr == nil && n > 0 {
			initCap = n
			if limit > 0 && initCap > limit {
				initCap = limit
			}
		}
	}
	if initCap > 0 {
		body, err = elton.ReadAllInitCap(r, initCap)
	} else {
		b := rr.bufferPool.Get()
		b.Reset()
		err = elton.ReadAllToBuffer(r, b)
		// 复制数据，因为buffer会继续复用
		body = append([]byte(nil), b.Bytes()...)
		// 当使用完时，buffer重新放入pool中
		rr.bufferPool.Put(b)
	}
	if err != nil {
		// 如果已经是http error
		if hes.Is(err) {
			return nil, err
		}
		// IO 读取失败的认为是 exception（hes.Wrap 默认 500 + Exception）
		return nil, hes.Wrap(err, hes.WithCategory(ErrBodyParserCategory))
	}
	return body, nil
}

// NewBodyParser returns a new body parser middleware.
// If limit < 0, it will be no limit for the body data.
// If limit = 0, it will use the default limit(50KB) for the body data.
// JSON content type validate is the default content validate function.
func NewBodyParser(config BodyParserConfig) elton.Handler {
	limit := defaultRequestBodyLimit
	if config.Limit != 0 {
		limit = config.Limit
	}
	initCap := 512
	if config.InitCap != 0 {
		initCap = config.InitCap
	}
	skipper := getSkipper(config.Skipper)
	contentTypeValidate := config.ContentTypeValidate
	if contentTypeValidate == nil {
		contentTypeValidate = DefaultJSONContentTypeValidate
	}
	// requestBodyReader无可变状态，可安全并发复用
	bodyReader := &requestBodyReader{
		bufferPool: elton.NewBufferPool(initCap),
		limit:      limit,
	}
	return func(c *elton.Context) error {
		if skipper(c) || c.RequestBody != nil || !contentTypeValidate(c) {
			return c.Next()
		}
		// 对于非提交数据的method跳过
		if !slices.Contains(validMethods, c.Request.Method) {
			return c.Next()
		}
		body, err := bodyReader.ReadAll(c)
		if err != nil {
			return err
		}
		c.RequestBody = body

		// 是否有设置on before decode
		if config.OnBeforeDecode != nil {
			err := config.OnBeforeDecode(c)
			if err != nil {
				return err
			}
		}

		matchDecoders := make([]BodyDecoder, 0)
		for _, decoder := range config.Decoders {
			if decoder.Validate(c) {
				matchDecoders = append(matchDecoders, decoder)
			}
		}
		// 没有符合条件的解码
		if len(matchDecoders) == 0 {
			return c.Next()
		}

		for _, decoder := range matchDecoders {
			body, err = decoder.Decode(c, body)
			if err != nil {
				return err
			}
		}
		c.RequestBody = body

		return c.Next()
	}
}
