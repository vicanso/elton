// MIT License

// Copyright (c) 2023 Tree Xie

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
	"context"
	"encoding/binary"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

var _ CacheStore = (*lruStore)(nil)

type lruStore struct {
	usePeek bool
	store   *lru.Cache[string, []byte]
}

const expiredByteSize = 4

func now_seconds() uint32 {
	return uint32(time.Now().Unix())
}

func (s *lruStore) Get(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	var ok bool
	// 使用peek更高性能
	if s.usePeek {
		value, ok = s.store.Peek(key)
	} else {
		value, ok = s.store.Get(key)
	}
	if !ok || len(value) < expiredByteSize {
		return nil, nil
	}
	expired := binary.BigEndian.Uint32(value)
	if now_seconds() > expired {
		return nil, nil
	}
	return value[expiredByteSize:], nil
}

func (s *lruStore) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	buf := make([]byte, len(data)+expiredByteSize)
	expired := now_seconds() + uint32(ttl/time.Second)
	binary.BigEndian.PutUint32(buf, expired)
	copy(buf[expiredByteSize:], data)
	s.store.Add(key, buf)
	return nil
}

func newLruStore(size int, usePeek bool) *lruStore {
	if size <= 0 {
		size = 128
	}
	// 只要size > 0则不会出错
	s, _ := lru.New[string, []byte](size)
	return &lruStore{
		usePeek: usePeek,
		store:   s,
	}
}

// NewPeekLruStore creates a lru store use peek
func NewPeekLruStore(size int) *lruStore {
	return newLruStore(size, true)
}

// NewLruStore creates a lru store
func NewLruStore(size int) *lruStore {
	return newLruStore(size, false)
}
