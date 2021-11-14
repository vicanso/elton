// MIT License

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
	"bytes"
	"context"
	"encoding/binary"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vicanso/elton"
)

type CacheConfig struct {
	// Skipper skipper function
	Skipper elton.Skipper
	// Store cache store
	Store CacheStore
	// HitForPassTTL hit for pass ttl
	HitForPassTTL time.Duration
}

type CacheStatus uint8

const (
	// StatusUnknown unknown status
	StatusUnknown CacheStatus = iota
	// StatusHitForPass hit-for-pass status
	StatusHitForPass
	// StatusHit hit cache status
	StatusHit
)

type CacheStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
}

const HeaderAge = "Age"
const HeaderXCache = "X-Cache"

var ignoreHeaders = []string{
	"Content-Encoding",
	"Content-Length",
	"Connection",
	"Date",
	HeaderXCache,
}

var (
	noCacheReg = regexp.MustCompile(`no-cache|no-store|private`)
	sMaxAgeReg = regexp.MustCompile(`s-maxage=(\d+)`)
	maxAgeReg  = regexp.MustCompile(`max-age=(\d+)`)
)

// IsPassCacheMethod is the method pass cache
func IsPassCacheMethod(reqMethod string) bool {
	if reqMethod != http.MethodGet && reqMethod != http.MethodHead {
		return true
	}
	return false
}

func isCacheable(c *elton.Context) (bool, int) {
	// 如果有content-encoding，不缓存
	if c.GetHeader(elton.HeaderContentEncoding) != "" {
		return false, -2
	}
	// 如果未设置数据
	if c.StatusCode == 0 && c.BodyBuffer == nil {
		return false, -1
	}
	// 不可缓存
	cacheAge := GetCacheMaxAge(c.Header())
	return cacheAge > 0, cacheAge
}

// GetCacheMaxAge returns the age of cache,
// get value from cache-control(s-maxage or max-age)
func GetCacheMaxAge(header http.Header) int {
	// 如果有设置cookie，则为不可缓存
	if header.Get(elton.HeaderSetCookie) != "" {
		return 0
	}
	// 如果没有设置cache-control，则不可缓存
	cacheControlKey := strings.ToLower(elton.HeaderCacheControl)
	var cc string
	for k, v := range header {
		if elton.HeaderCacheControl == k || strings.ToLower(k) == cacheControlKey {
			cc = strings.Join(v, ",")
			break
		}
	}
	if cc == "" {
		return 0
	}

	// 如果设置不可缓存，返回0
	if noCacheReg.MatchString(cc) {
		return 0
	}
	// 优先从s-maxage中获取
	var maxAge = 0
	result := sMaxAgeReg.FindStringSubmatch(cc)
	if len(result) == 2 {
		maxAge, _ = strconv.Atoi(result[1])
	} else {
		// 从max-age中获取缓存时间
		result = maxAgeReg.FindStringSubmatch(cc)
		if len(result) == 2 {
			maxAge, _ = strconv.Atoi(result[1])
		}
	}

	// 如果有设置了 age 字段，则最大缓存时长减少
	if age := header.Get(HeaderAge); age != "" {
		v, _ := strconv.Atoi(age)
		maxAge -= v
	}

	return maxAge
}

const (
	// 状态字节数
	statusByteSize = 1
	// 创建时间保存的字节数
	createAtByteSize = 4
	// 状态码保存的字节数
	statusCodeByteSize = 2
	// 保存请求头长度的字节数
	headerBytesSize = 4
	// 压缩类型的字节数
	compressionBytesSize = 1
)

// 数据结构[状态(1字节), 创建时间(4字节), 状态码(2字节), 请求头长度(4字节), 请求头内容(N字节), 压缩类型(1字节) 响应内容(N字节)]
type CacheResponse struct {
	Status CacheStatus
	// 创建时间
	CreatedAt uint32
	// 状态码
	StatusCode int
	// 响应头
	Header http.Header
	// 压缩类型
	Compression CompressionType
	// 响应数据
	Body *bytes.Buffer
}

var hitForPassData = (&CacheResponse{
	Status: StatusHitForPass,
}).Bytes()

// Bytes converts the cache response to bytes
func (cp *CacheResponse) Bytes() []byte {
	// 只有hit的才需要保存后续的数据
	if cp.Status != StatusHit {
		return []byte{
			byte(cp.Status),
		}
	}
	headers := NewHTTPHeaders(cp.Header, ignoreHeaders...)
	headersLength := len(headers)
	// 4个字节保存创建时间
	// 2个字节保存status code
	// 4个字节保存http header长度
	// http header数据
	// 1个字节保存compression type
	// body 数据
	bodySize := 0
	if cp.Body != nil {
		bodySize = cp.Body.Len()
	}

	size := statusByteSize + createAtByteSize + statusCodeByteSize + headerBytesSize + headersLength + compressionBytesSize + bodySize

	buf := make([]byte, size)
	offset := 0

	buf[offset] = byte(cp.Status)
	offset += statusByteSize

	binary.BigEndian.PutUint32(buf[offset:], cp.CreatedAt)
	offset += createAtByteSize

	binary.BigEndian.PutUint16(buf[offset:], uint16(cp.StatusCode))
	offset += statusCodeByteSize

	binary.BigEndian.PutUint32(buf[offset:], uint32(len(headers)))
	offset += headerBytesSize

	if headersLength != 0 {
		copy(buf[offset:], headers)
		offset += headersLength
	}

	buf[offset] = byte(cp.Compression)
	offset += compressionBytesSize

	if bodySize != 0 {
		copy(buf[offset:], cp.Body.Bytes())
	}

	return buf
}

// GetBody returns the data match accept encoding
func (cp *CacheResponse) GetBody(acceptEncoding string) (*bytes.Buffer, string, error) {

	for compressType, decompressor := range cacheDecompressors {
		// comporession match decompressor
		if cp.Compression == compressType {
			encoding := decompressor.GetEncoding()
			// client acccept the encoding
			if strings.Contains(acceptEncoding, encoding) {
				return cp.Body, encoding, nil
			}
			// decompress
			data, err := decompressor.Decompress(cp.Body)
			if err != nil {
				return nil, "", err
			}
			return data, "", nil
		}
	}
	return cp.Body, "", nil
}

// SetBody sets body to context, it will be matched acccept-encoding
func (cp *CacheResponse) SetBody(c *elton.Context) error {
	// 如果body不为空
	if cp.Body != nil && cp.Body.Len() != 0 {
		body, encoding, err := cp.GetBody(c.GetRequestHeader(elton.HeaderAcceptEncoding))
		if err != nil {
			return err
		}
		c.SetHeader(elton.HeaderContentEncoding, encoding)
		c.BodyBuffer = body
	}
	return nil
}

// NewCacheResponse create a new cache response
func NewCacheResponse(data []byte) *CacheResponse {
	dataSize := len(data)
	if dataSize < statusByteSize {
		return &CacheResponse{
			Status: StatusUnknown,
		}
	}
	if dataSize < statusByteSize+statusCodeByteSize+headerBytesSize {
		return &CacheResponse{
			Status: CacheStatus(data[0]),
		}
	}
	offset := 0

	status := data[offset]
	offset += statusByteSize

	createdAt := binary.BigEndian.Uint32(data[offset:])
	offset += createAtByteSize

	statusCode := binary.BigEndian.Uint16(data[offset:])
	offset += statusCodeByteSize

	size := int(binary.BigEndian.Uint32(data[offset:]))
	offset += headerBytesSize

	hs := HTTPHeaders(data[offset : offset+size])
	offset += size

	commpression := data[offset]
	offset += compressionBytesSize

	body := data[offset:]

	return &CacheResponse{
		Status:      CacheStatus(status),
		CreatedAt:   createdAt,
		StatusCode:  int(statusCode),
		Header:      hs.Header(),
		Compression: CompressionType(commpression),
		Body:        bytes.NewBuffer(body),
	}
}

// NewCache return a new cache middleware.
func NewCache(config CacheConfig) elton.Handler {
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	store := config.Store
	if store == nil {
		panic("require store for cache")
	}
	hitForPassTTL := 5 * time.Minute
	if config.HitForPassTTL > 0 {
		hitForPassTTL = config.HitForPassTTL
	}
	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		method := c.Request.Method
		if IsPassCacheMethod(method) {
			return c.Next()
		}
		ctx := c.Context()
		key := c.Request.Host + " " + method + " " + c.Request.RequestURI
		data, err := store.Get(ctx, key)
		if err != nil {
			return err
		}
		cacheResp := NewCacheResponse(data)
		switch cacheResp.Status {
		// 如果是hit for pass，直接返回
		case StatusHitForPass:
			c.SetHeader(HeaderXCache, "hit-for-pass")
			return c.Next()
		// 如果获取到数据，则直接响应，不需要next
		case StatusHit:
			c.SetHeader(HeaderXCache, "hit")
			age := uint32(time.Now().Unix()) - cacheResp.CreatedAt
			c.SetHeader(HeaderAge, strconv.Itoa(int(age)))
			c.StatusCode = cacheResp.StatusCode
			c.MergeHeader(cacheResp.Header)
			cacheResp.SetBody(c)
			return nil
		}
		c.SetHeader(HeaderXCache, "fetch")
		err = c.Next()
		if err != nil {
			return err
		}
		cacheable, cacheAge := isCacheable(c)
		// 不可缓存
		if !cacheable {
			// 对于fetch请求，不能缓存的设置hit for pass
			return store.Set(ctx, key, hitForPassData, hitForPassTTL)
		}

		// TODO 数据压缩
		cacheResp = &CacheResponse{
			// 状态设置为hit
			Status:     StatusHit,
			CreatedAt:  uint32(time.Now().Unix()),
			StatusCode: c.StatusCode,
			Header:     c.Header(),
			Body:       c.BodyBuffer,
		}
		data = cacheResp.Bytes()
		// 如果想忽略store的错误，则自定义store时，
		// 不要返回出错则可
		err = store.Set(ctx, key, data, time.Duration(cacheAge)*time.Second)
		if err != nil {
			return err
		}
		return cacheResp.SetBody(c)
	}
}
