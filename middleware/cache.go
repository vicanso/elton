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
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/vicanso/elton/v2"
)

type CacheConfig struct {
	// Skipper skipper function
	Skipper elton.Skipper
	// Store cache store
	Store CacheStore
	// HitForPassTTL hit for pass ttl
	HitForPassTTL time.Duration
	// Compressor cache compressor
	Compressor CacheCompressor
	// GetKey get the key for request
	GetKey func(*elton.Context) string
	// Marshal marshal function for cache, if BodyBuffer is nil,
	// the body will be marshaled to body buffer. The default marshal function will be json.Marshal
	Marshal func(any) ([]byte, error)
	// IgnoreHeaders ignore the headers for cache
	IgnoreHeaders []string
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

// defaultIgnoreHeaders 序列化缓存响应时默认忽略的响应头：
// Content-Encoding/Content-Length 由缓存的压缩类型与body重新生成，
// Date/Age/X-Cache 每次命中时重新计算，Connection 属于连接级头不可缓存
var defaultIgnoreHeaders = []string{
	"Content-Encoding",
	"Content-Length",
	"Connection",
	"Date",
	HeaderAge,
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
	// 如果是流，则不缓存
	if c.IsReaderBody() {
		return false, -2
	}
	// 如果有content-encoding，不缓存
	if c.GetHeader(elton.HeaderContentEncoding) != "" {
		return false, -2
	}
	// 如果未设置数据
	if c.StatusCode == 0 &&
		c.BodyBuffer == nil &&
		c.Body == nil {
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
	cc := strings.Join(header.Values(elton.HeaderCacheControl), ",")
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
	// 缓存状态字节数
	statusByteSize = 1
	// 创建时间保存的字节数(单位秒)
	createAtByteSize = 4
	// 状态码保存的字节数
	statusCodeByteSize = 2
	// 保存请求头长度的字节数
	headerBytesSize = 4
	// 响应数据压缩类型的字节数
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
	// 响应数据压缩类型
	Compression CompressionType
	// 响应数据
	Body *bytes.Buffer
}

// hitForPassData hit-for-pass状态的预编码数据（仅1个状态字节），
// 所有不可缓存的URL共用此份数据
var hitForPassData = (&CacheResponse{
	Status: StatusHitForPass,
}).Bytes()

// Bytes converts the cache response to bytes
func (cp *CacheResponse) Bytes(ignoreHeaders ...string) []byte {
	// 只有hit的才需要保存后续的数据
	if cp.Status != StatusHit {
		return []byte{
			byte(cp.Status),
		}
	}
	// 拼接为新的slice，避免对入参底层数组的并发写
	ignoreHeaders = slices.Concat(ignoreHeaders, defaultIgnoreHeaders)
	headers := NewHTTPHeaders(cp.Header, ignoreHeaders...)
	headersLength := len(headers)
	// 1个字节保存状态
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
func (cp *CacheResponse) GetBody(acceptEncoding string, compressor CacheCompressor) (*bytes.Buffer, string, error) {
	if compressor != nil && cp.Compression == compressor.Compression() {
		encoding := compressor.Encoding()
		// client accept the encoding
		if strings.Contains(acceptEncoding, encoding) {
			return cp.Body, encoding, nil
		}
		// decompress
		data, err := compressor.Decompress(cp.Body)
		if err != nil {
			return nil, "", err
		}
		return data, "", nil
	}
	return cp.Body, "", nil
}

// SetBody sets body to http response, it will be matched accept-encoding
func (cp *CacheResponse) SetBody(c *elton.Context, compressor CacheCompressor) error {
	// 如果body不为空
	if cp.Body != nil && cp.Body.Len() != 0 {
		body, encoding, err := cp.GetBody(c.GetRequestHeader(elton.HeaderAcceptEncoding), compressor)
		if err != nil {
			return err
		}
		c.SetHeader(elton.HeaderContentEncoding, encoding)
		c.BodyBuffer = body
	}
	return nil
}

// NewCacheResponse decodes the cache data to cache response,
// it's the reverse operation of CacheResponse.Bytes.
// 数据布局: [状态(1字节)][创建时间(4字节)][状态码(2字节)][请求头长度(4字节)][请求头内容(N字节)][压缩类型(1字节)][响应内容(N字节)]
// 对于hit-for-pass的缓存，只有状态字节。
// 若数据不完整（如自定义store返回损坏数据），返回StatusUnknown，按fetch处理。
func NewCacheResponse(data []byte) *CacheResponse {
	dataSize := len(data)
	// 无数据或无状态字节
	if dataSize < statusByteSize {
		return &CacheResponse{
			Status: StatusUnknown,
		}
	}
	// 定长头部（状态+创建时间+状态码+请求头长度）共11字节，
	// 不足时仅返回状态（hit-for-pass只写入状态字节）
	prefixSize := statusByteSize + createAtByteSize + statusCodeByteSize + headerBytesSize
	if dataSize < prefixSize {
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

	// 请求头长度来自数据本身，需校验剩余数据长度，
	// 避免损坏数据导致越界
	if dataSize < offset+size+compressionBytesSize {
		return &CacheResponse{
			Status: StatusUnknown,
		}
	}

	hs := HTTPHeaders(data[offset : offset+size])
	offset += size

	compression := data[offset]
	offset += compressionBytesSize

	body := data[offset:]

	return &CacheResponse{
		Status:      CacheStatus(status),
		CreatedAt:   createdAt,
		StatusCode:  int(statusCode),
		Header:      hs.Header(),
		Compression: CompressionType(compression),
		Body:        bytes.NewBuffer(body),
	}
}

// NewDefaultCache return a new cache middleware with brotli compress
func NewDefaultCache(store CacheStore) elton.Handler {
	return NewCache(CacheConfig{
		Store:      store,
		Compressor: NewCacheBrCompressor(),
	})
}

func cacheDefaultGetKey(c *elton.Context) string {
	// 默认处理不添加host
	return c.Request.Method + " " + c.Request.RequestURI
}

func getBodyBuffer(c *elton.Context, marshal func(any) ([]byte, error)) (*bytes.Buffer, error) {
	if c.BodyBuffer != nil {
		return c.BodyBuffer, nil
	}
	buf, err := marshal(c.Body)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(buf), nil
}

// NewCache return a new cache middleware.
//
// 缓存为三态状态机：
//   - fetch: 缓存无数据，执行后续中间件获取响应。若响应可缓存
//     （Cache-Control 带 max-age/s-maxage 且无 Set-Cookie），则以 hit 状态存储；
//     否则存储 hit-for-pass 标记（TTL为HitForPassTTL）。
//   - hit: 命中缓存，直接以缓存数据响应，不再执行后续中间件。
//   - hit-for-pass: 此前已确认该URL响应不可缓存，直接透传至后续中间件，
//     避免在TTL内反复尝试缓存判定。
//
// 仅 GET/HEAD 请求走缓存逻辑，其它method直接透传。
func NewCache(config CacheConfig) elton.Handler {
	skipper := getSkipper(config.Skipper)
	store := config.Store
	if store == nil {
		panic("require store for cache")
	}
	hitForPassTTL := 5 * time.Minute
	if config.HitForPassTTL > 0 {
		hitForPassTTL = config.HitForPassTTL
	}
	getKey := config.GetKey
	if getKey == nil {
		getKey = cacheDefaultGetKey
	}
	marshal := config.Marshal
	if marshal == nil {
		marshal = json.Marshal
	}
	ignoreHeaders := config.IgnoreHeaders
	compressor := config.Compressor
	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		if IsPassCacheMethod(c.Request.Method) {
			return c.Next()
		}
		ctx := c.Context()
		key := getKey(c)
		data, err := store.Get(ctx, key)
		if err != nil {
			return err
		}
		cacheResp := NewCacheResponse(data)
		switch cacheResp.Status {
		// 如果是hit for pass，直接转至后续中间件
		case StatusHitForPass:
			c.SetHeader(HeaderXCache, "hit-for-pass")
			return c.Next()
		// 如果获取到数据，则直接响应，不需要next转至后续中间件
		case StatusHit:
			c.SetHeader(HeaderXCache, "hit")
			age := uint32(time.Now().Unix()) - cacheResp.CreatedAt
			c.SetHeader(HeaderAge, strconv.Itoa(int(age)))
			c.StatusCode = cacheResp.StatusCode
			// 要先清除原有的响应头中的Cache-Control
			c.SetHeader(elton.HeaderCacheControl, "")
			c.MergeHeader(cacheResp.Header)
			return cacheResp.SetBody(c, compressor)
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

		buffer, err := getBodyBuffer(c, marshal)
		if err != nil {
			return err
		}
		compressionType := CompressionNone
		if compressor != nil &&
			compressor.IsValid(c.GetHeader(elton.HeaderContentType), buffer.Len()) {
			// 符合压缩条件
			buffer, compressionType, err = compressor.Compress(buffer)
			if err != nil {
				return err
			}
		}

		cacheResp = &CacheResponse{
			// 状态设置为hit
			Status:      StatusHit,
			Compression: compressionType,
			CreatedAt:   uint32(time.Now().Unix()),
			StatusCode:  c.StatusCode,
			Header:      c.Header(),
			Body:        buffer,
		}
		data = cacheResp.Bytes(ignoreHeaders...)
		// 如果想忽略store的错误，则自定义store时，
		// 不要返回出错则可
		err = store.Set(ctx, key, data, time.Duration(cacheAge)*time.Second)
		if err != nil {
			return err
		}
		return cacheResp.SetBody(c, compressor)
	}
}
