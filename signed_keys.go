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

package cod

import "sync"

type (
	// SignedKeysGenerator signed keys generator
	SignedKeysGenerator interface {
		GetKeys() []string
		SetKeys([]string)
	}
	// SimpleSignedKeys simple sigined key
	SimpleSignedKeys struct {
		keys []string
	}
	// RWMutexSignedKeys read/write mutex signed key
	RWMutexSignedKeys struct {
		sync.RWMutex
		keys []string
	}
)

// GetKeys get keys
func (sk *SimpleSignedKeys) GetKeys() []string {
	return sk.keys
}

// SetKeys set keys
func (sk *SimpleSignedKeys) SetKeys(keys []string) {
	sk.keys = keys
}

// GetKeys get keys
func (rwSk *RWMutexSignedKeys) GetKeys() []string {
	rwSk.RLock()
	defer rwSk.RUnlock()
	return rwSk.keys
}

// SetKeys set keys
func (rwSk *RWMutexSignedKeys) SetKeys(keys []string) {
	rwSk.Lock()
	defer rwSk.Unlock()
	rwSk.keys = keys
}
