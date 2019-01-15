package middleware

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/cod"
)

func TestTracker(t *testing.T) {
	done := false
	fn := NewTracker(TrackerConfig{
		OnTrack: func(info *TrackerInfo, _ *cod.Context) {
			if info.Result != HandleFail ||
				info.Query["type"] != "1" ||
				info.Query["passwordType"] != "***" ||
				info.Params["category"] != "login" ||
				info.Form["password"] != "***" {
				t.Fatalf("tracker info is invalid")
			}
			done = true
		},
	})
	req := httptest.NewRequest("POST", "/users/login?type=1&passwordType=2", nil)
	c := cod.NewContext(nil, req)
	c.RequestBody = []byte(`{
		"account": "tree.xie",
		"password": "password"
	}`)
	c.Params = map[string]string{
		"category": "login",
	}
	c.Next = func() error {
		return errors.New("abcd")
	}
	fn(c)
	if !done {
		t.Fatalf("tracker middleware fail")
	}
}
