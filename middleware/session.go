// Copyright 2018 tree xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package middleware

import (
	"net/http"
	"time"

	"github.com/spf13/cast"
	"github.com/vicanso/cod"
	"github.com/vicanso/hes"
)

const (
	// CreatedAt the created time for session
	CreatedAt = "_createdAt"
	// UpdatedAt the updated time for session
	UpdatedAt = "_updatedAt"
)

var (
	// errNotFetched not fetch error
	errNotFetched = &hes.Error{
		Message:    "not fetch session",
		Category:   ErrCategorySession,
		StatusCode: http.StatusInternalServerError,
	}
	// errResetSessionID reset session id fail
	errResetSessionID = &hes.Error{
		Message:    "reset session id fail",
		Category:   ErrCategorySession,
		StatusCode: http.StatusInternalServerError,
	}
	// errDuplicateCommit duplicate commit
	errDuplicateCommit = &hes.Error{
		Message:    "duplicate commit",
		Category:   ErrCategorySession,
		StatusCode: http.StatusInternalServerError,
	}
)

type (
	// SessionConfig session config
	SessionConfig struct {
		CreateStore func(c *cod.Context) (Store, error)
		Skipper     Skipper
	}
	// Store session store interface
	Store interface {
		// GetID get the session id
		GetID() string
		// CreateID create session id
		CreateID() (string, error)
		// Get get the session data
		Get(string) ([]byte, error)
		// Set set the session data
		Set(string, []byte) error
		// Destroy remove the session data
		Destroy(string) error
	}
	// Session session struct
	Session struct {
		// Store session store
		Store Store
		id    string
		// the data fetch from session
		data cod.M
		// the data has been fetched
		fetched bool
		// the data has been modified
		modified bool
		// the session has been committed
		committed bool
	}
)

func getInitMap() cod.M {
	m := make(cod.M)
	m[CreatedAt] = time.Now().Format(time.RFC3339)
	return m
}

// Fetch fetch the session data from store
func (s *Session) Fetch() (m cod.M, err error) {
	if s.fetched {
		m = s.data
		return
	}
	store := s.Store

	value := store.GetID()
	s.id = value
	var buf []byte
	if value != "" {
		buf, err = store.Get(value)
		if err != nil {
			return
		}
	}
	m = make(cod.M)
	if len(buf) == 0 {
		m = getInitMap()
	} else {
		err = json.Unmarshal(buf, &m)
	}
	if err != nil {
		return
	}
	s.fetched = true
	s.data = m
	return
}

// Destroy remove the data from store and reset session data
func (s *Session) Destroy() (err error) {
	store := s.Store
	value := store.GetID()
	m := getInitMap()
	s.data = m

	if value == "" {
		return
	}
	err = store.Destroy(value)
	return
}

// Set set data to session
func (s *Session) Set(key string, value interface{}) (err error) {
	if key == "" {
		return
	}
	if !s.fetched {
		return errNotFetched
	}
	if value == nil {
		delete(s.data, key)
	} else {
		s.data[key] = value
	}
	s.data[UpdatedAt] = time.Now().Format(time.RFC3339)
	s.modified = true
	return
}

// SetMap set map data to session
func (s *Session) SetMap(value map[string]interface{}) (err error) {
	if value == nil {
		return
	}
	if !s.fetched {
		return errNotFetched
	}
	for k, v := range value {
		if v == nil {
			delete(s.data, k)
			continue
		}
		s.data[k] = v
	}

	s.data[UpdatedAt] = time.Now().Format(time.RFC3339)
	s.modified = true
	return
}

// Refresh refresh session (update updatedAt)
func (s *Session) Refresh() (err error) {
	if !s.fetched {
		return errNotFetched
	}
	s.data[UpdatedAt] = time.Now().Format(time.RFC3339)
	s.modified = true
	return
}

// Get get data from session's data
func (s *Session) Get(key string) interface{} {
	if !s.fetched {
		return nil
	}
	return s.data[key]
}

// GetBool get bool data from session's data
func (s *Session) GetBool(key string) bool {
	if !s.fetched {
		return false
	}
	return cast.ToBool(s.data[key])
}

// GetString get string data from session's data
func (s *Session) GetString(key string) string {
	if !s.fetched {
		return ""
	}
	return cast.ToString(s.data[key])
}

// GetInt get int data from session's data
func (s *Session) GetInt(key string) int {
	if !s.fetched {
		return 0
	}
	return cast.ToInt(s.data[key])
}

// GetFloat64 get float64 data from session's data
func (s *Session) GetFloat64(key string) float64 {
	if !s.fetched {
		return 0
	}
	return cast.ToFloat64(s.data[key])
}

// GetStringSlice get string slice data from session's data
func (s *Session) GetStringSlice(key string) []string {
	if !s.fetched {
		return nil
	}
	return cast.ToStringSlice(s.data[key])
}

// GetCreatedAt get the created at of session
func (s *Session) GetCreatedAt() string {
	if !s.fetched {
		return ""
	}
	v := s.data[CreatedAt]
	if v == nil {
		return ""
	}
	return v.(string)
}

// GetUpdatedAt get the updated at of session
func (s *Session) GetUpdatedAt() string {
	if !s.fetched {
		return ""
	}
	v := s.data[UpdatedAt]
	if v == nil {
		return ""
	}
	return v.(string)
}

// GetData get the session's data
func (s *Session) GetData() cod.M {
	return s.data
}

// ResetSessionID reset session id
func (s *Session) ResetSessionID() (string, error) {
	return s.Store.CreateID()
}

// GetID get session id
func (s *Session) GetID() string {
	return s.id
}

// Commit sync the session to store
func (s *Session) Commit() (err error) {
	if !s.modified {
		return
	}
	if s.committed {
		err = errDuplicateCommit
		return
	}
	// 如果session id为空，则生成新的session id
	if s.id == "" {
		newID, e := s.ResetSessionID()
		if e != nil {
			err = e
			return
		}
		if newID == "" {
			err = errResetSessionID
			return
		}
		s.id = newID
	}

	buf, err := json.Marshal(s.data)
	if err != nil {
		return
	}
	err = s.Store.Set(s.id, buf)
	if err != nil {
		return
	}
	s.committed = true
	return
}

// NewSession create a new session middleware
func NewSession(config SessionConfig) cod.Handler {
	if config.CreateStore == nil {
		panic("require create store function")
	}
	skipper := config.Skipper
	if skipper == nil {
		skipper = DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		store, err := config.CreateStore(c)
		if err != nil {
			return
		}
		s := &Session{
			Store: store,
		}
		_, err = s.Fetch()
		if err != nil {
			return
		}
		c.Set(cod.SessionKey, s)
		err = c.Next()
		if err != nil {
			return
		}
		err = s.Commit()
		return
	}
}
