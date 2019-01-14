package middleware

import (
	"errors"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/vicanso/cod"
)

func TestConcurrentLimiter(t *testing.T) {
	m := new(sync.Map)
	fn := NewConcurrentLimiter(ConcurrentLimiterConfig{
		Keys: []string{
			":ip",
			"h:X-Token",
			"q:type",
			"p:id",
			"account",
		},
		Lock: func(key string, c *cod.Context) (success bool, unlock func(), err error) {
			if key != "192.0.2.1,xyz,1,123,tree.xie" {
				err = errors.New("key is invalid")
				return
			}
			_, loaded := m.LoadOrStore(key, 1)
			// 如果已存在，则获取销失败
			if loaded {
				return
			}
			success = true
			// 删除锁
			unlock = func() {
				m.Delete(key)
			}
			return
		},
	})

	req := httptest.NewRequest("POST", "/users/login?type=1", nil)
	resp := httptest.NewRecorder()
	c := cod.NewContext(resp, req)
	req.Header.Set("X-Token", "xyz")
	c.RequestBody = []byte(`{
		"account": "tree.xie"
	}`)
	c.Params = map[string]string{
		"id": "123",
	}

	t.Run("first", func(t *testing.T) {
		done := false
		c.Next = func() error {
			done = true
			return nil
		}
		err := fn(c)
		// 登录限制,192.0.2.1,xyz,1,123,tree.xie
		if err != nil || !done {
			t.Fatalf("concurrent limiter fail, %v", err)
		}
	})

	t.Run("too frequently", func(t *testing.T) {
		done := false
		c.Next = func() error {
			time.Sleep(100 * time.Millisecond)
			done = true
			return nil
		}
		go func() {
			time.Sleep(10 * time.Millisecond)
			e := fn(c)
			if e.Error() != "category=cod-concurrent-limiter, message=submit too frequently" {
				t.Fatalf("request should return too frequently")
			}
		}()
		err := fn(c)
		// 登录限制,192.0.2.1,xyz,1,123,tree.xie
		if err != nil || !done {
			t.Fatalf("concurrent limiter fail, %v", err)
		}
	})

	t.Run("lock function return error", func(t *testing.T) {
		c.Params = map[string]string{}
		err := fn(c)
		if err.Error() != "key is invalid" {
			t.Fatalf("lock function should return error")
		}
	})

}
