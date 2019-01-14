package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/hes"

	"github.com/vicanso/cod"
)

func TestStats(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://127.0.0.1/users/me", nil)
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		c.BodyBuffer = bytes.NewBufferString("abcd")
		done := false
		fn := NewStats(StatsConfig{
			OnStats: func(info *StatsInfo, _ *cod.Context) {
				if info.Status != http.StatusOK {
					t.Fatalf("status code should be 200")
				}
				done = true
			},
		})
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		if err != nil {
			t.Fatalf("stats middleware fail, %v", err)
		}
		if !done {
			t.Fatalf("on stats is not called")
		}
	})

	t.Run("return error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://127.0.0.1/users/me", nil)
		resp := httptest.NewRecorder()
		c := cod.NewContext(resp, req)
		done := false
		fn := NewStats(StatsConfig{
			OnStats: func(info *StatsInfo, _ *cod.Context) {
				if info.Status != http.StatusBadRequest {
					t.Fatalf("status code should be 400")
				}
				done = true
			},
		})
		c.Next = func() error {
			return hes.New("abc")
		}
		fn(c)
		if !done {
			t.Fatalf("on stats is not called")
		}
	})
}
