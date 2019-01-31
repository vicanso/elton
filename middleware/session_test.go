package middleware

import (
	"math/rand"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/vicanso/cod"
)

type (
	TmpStore struct {
		id   string
		data []byte
		ctx  *cod.Context
	}
)

const (
	sessionKey = "X-Session-ID"
)

func (s *TmpStore) GetID() string {
	ctx := s.ctx
	if ctx != nil {
		return ctx.GetHeader(sessionKey)
	}
	return s.id
}
func (s *TmpStore) CreateID() (string, error) {
	ctx := s.ctx
	s.id = strconv.Itoa(rand.Int())
	if ctx != nil {
		ctx.SetHeader(sessionKey, s.id)
	}
	return s.id, nil
}
func (s *TmpStore) Get(id string) ([]byte, error) {
	return s.data, nil
}
func (s *TmpStore) Set(id string, data []byte) error {
	s.data = data
	return nil
}
func (s *TmpStore) Destroy(id string) error {
	s.data = nil
	return nil
}

func TestFetch(t *testing.T) {
	store := &TmpStore{}
	s := Session{
		Store: store,
	}
	m, err := s.Fetch()
	if err != nil || m == nil || m["_createdAt"].(string) == "" {
		t.Fatalf("fetch fail, %v", err)
	}
	m, err = s.Fetch()
	if err != nil || m == nil || m["_createdAt"].(string) == "" {
		t.Fatalf("fetch fail, %v", err)
	}
}

func TestSet(t *testing.T) {
	store := &TmpStore{}
	s := Session{
		Store: store,
	}
	// empty key should pass
	err := s.Set("", "b")
	if err != nil {
		t.Fatalf("set fail, %v", err)
	}
	err = s.Set("a", "b")
	if err != errNotFetched {
		t.Fatalf("should return not fetched error")
	}
	s.Fetch()
	s.Set("a", "b")
	if s.GetString("a") != "b" {
		t.Fatalf("set fail")
	}
	s.Set("a", nil)
	if s.GetString("a") != "" {
		t.Fatalf("unset fail")
	}
}

func TestSetMap(t *testing.T) {
	store := &TmpStore{}
	s := Session{
		Store: store,
	}
	// empty key should pass
	err := s.SetMap(nil)
	if err != nil {
		t.Fatalf("set fail, %v", err)
	}
	err = s.SetMap(map[string]interface{}{
		"a": "b",
	})
	if err != errNotFetched {
		t.Fatalf("should return not fetched error")
	}
	s.Fetch()
	err = s.SetMap(map[string]interface{}{
		"a": "b",
	})
	if err != nil || s.GetString("a") != "b" {
		t.Fatalf("set map fail, %v", err)
	}
	err = s.SetMap(map[string]interface{}{
		"a": nil,
	})
	if err != nil || s.GetString("a") != "" {
		t.Fatalf("unset map fail, %v", err)
	}
}

func TestRefresh(t *testing.T) {
	store := &TmpStore{}
	s := Session{
		Store: store,
	}
	err := s.Refresh()
	if err != errNotFetched {
		t.Fatalf("should return not fetched error")
	}
	s.Fetch()
	err = s.Refresh()
	if err != nil || !s.modified {
		t.Fatalf("refresh fail, %v", err)
	}
}

func TestGet(t *testing.T) {
	store := &TmpStore{}
	s := Session{
		Store: store,
	}

	if s.Get("i") != nil {
		t.Fatalf("get before fetch fail")
	}

	if s.GetBool("b") {
		t.Fatalf("get bool before fetch fail")
	}

	if s.GetString("s") != "" {
		t.Fatalf("get string before fetch fail")
	}

	if s.GetInt("i") != 0 {
		t.Fatalf("get int before fetch fail")
	}

	if s.GetFloat64("f") != 0 {
		t.Fatalf("get float before fetch fail")
	}

	if s.GetStringSlice("ss") != nil {
		t.Fatalf("get string slice before fetch fail")
	}

	if s.GetCreatedAt() != "" {
		t.Fatalf("get created at before fetch fail")
	}

	if s.GetUpdatedAt() != "" {
		t.Fatalf("get updated at before fetch fail")
	}

	if s.GetData() != nil {
		t.Fatalf("get data before fetch fail")
	}

	s.Fetch()
	s.SetMap(map[string]interface{}{
		"i": 1,
		"b": true,
		"s": "a",
		"f": 1.1,
		"ss": []string{
			"a",
			"b",
		},
	})
	if s.Get("i") == nil {
		t.Fatalf("get fail")
	}

	if !s.GetBool("b") {
		t.Fatalf("get bool fail")
	}

	if s.GetString("s") != "a" {
		t.Fatalf("get string fail")
	}

	if s.GetInt("i") != 1 {
		t.Fatalf("get int fail")
	}

	if s.GetFloat64("f") != 1.1 {
		t.Fatalf("get float fail")
	}

	if strings.Join(s.GetStringSlice("ss"), ",") != "a,b" {
		t.Fatalf("get string slice fail")
	}

	if s.GetCreatedAt() == "" {
		t.Fatalf("get created at fail")
	}

	if s.GetUpdatedAt() == "" {
		t.Fatalf("get updated at fail")
	}

	if s.GetData() == nil {
		t.Fatalf("get data fail")
	}

	s.Commit()

	if s.GetID() == "" {
		t.Fatalf("get session id fail")
	}

}

func TestResetSessionID(t *testing.T) {
	store := &TmpStore{}
	s := Session{
		Store: store,
	}
	originalID := s.Store.GetID()
	s.ResetSessionID()
	if s.Store.GetID() == originalID {
		t.Fatalf("session id should be recreated")
	}
}

func TestCommit(t *testing.T) {
	store := &TmpStore{}
	s := Session{
		Store: store,
	}
	s.Fetch()
	s.Set("a", "b")
	err := s.Commit()
	if err != nil {
		t.Fatalf("commit fail, %v", err)
	}
	err = s.Commit()
	if err != errDuplicateCommit {
		t.Fatalf("duplicate commit should return error")
	}
}

func TestDestroy(t *testing.T) {
	store := &TmpStore{}
	s := Session{
		Store: store,
	}
	store.CreateID()
	_, err := s.Fetch()
	if err != nil {
		t.Fatalf("fetch fail, %v", err)
	}
	s.Set("a", "b")
	if s.GetString("a") != "b" {
		t.Fatalf("set value fail")
	}
	err = s.Destroy()
	if err != nil || s.GetString("a") != "" {
		t.Fatalf("destroy fail, %v", err)
	}
}

func TestSession(t *testing.T) {
	createStore := func(c *cod.Context) Store {
		return &TmpStore{
			ctx: c,
		}
	}
	fn := NewSession(SessionConfig{
		CreateStore: createStore,
	})
	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/me", nil)
	c := cod.NewContext(resp, req)
	c.Next = func() error {
		se := c.Get(cod.SessionKey).(*Session)
		se.Set("account", "tree.xie")
		return nil
	}
	err := fn(c)
	if err != nil {
		t.Fatalf("session middleware fail, %v", err)
	}
	se := c.Get(cod.SessionKey).(*Session)
	if se.GetString("account") != "tree.xie" {
		t.Fatalf("set value to session fail")
	}
}
